package ratelimiter

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"ratelimit-challenge/internal/config"
	"ratelimit-challenge/pkg/ratelimiter"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

var _ = redis.Nil // Ensure redis package is imported

// Service provides rate limiting functionality with support for dynamic user limits
type Service struct {
	slidingWindow ratelimiter.RateLimiter
	leakyBucket   ratelimiter.RateLimiter
	config        *config.RateLimitConfig
	logger        *zap.Logger
	redisClient   *redis.Client

	// Local cache for user-specific rate limits
	// This reduces Redis lookups for frequently accessed users
	userLimitsCache map[string]int
	cacheMutex      sync.RWMutex
	cacheExpiry     map[string]time.Time
}

// NewService creates a new rate limiter service
func NewService(
	redisClient *redis.Client,
	cfg *config.RateLimitConfig,
	logger *zap.Logger,
) *Service {
	service := &Service{
		slidingWindow:   ratelimiter.NewSlidingWindow(redisClient, logger),
		leakyBucket:     ratelimiter.NewLeakyBucket(redisClient, logger),
		config:          cfg,
		logger:          logger,
		redisClient:     redisClient,
		userLimitsCache: make(map[string]int),
		cacheExpiry:     make(map[string]time.Time),
	}

	// Start cache cleanup goroutine
	if cfg.EnableLocalCache {
		go service.cleanupCache()
	}

	return service
}

// RateLimit checks if a request is allowed for a user
// This is the main function that should be called for each request
// It supports dynamic rate limits per user (stored in Redis)
func (s *Service) RateLimit(ctx context.Context, userID string, limit int) (bool, error) {
	// Get user-specific limit if configured, otherwise use provided limit
	userLimit, err := s.getUserLimit(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get user limit, using provided limit",
			zap.String("user_id", userID),
			zap.Int("fallback_limit", limit),
			zap.Error(err),
		)
		userLimit = limit
	}

	// Use configured limit if user limit not found
	if userLimit == 0 {
		userLimit = limit
	}

	windowSize := time.Duration(s.config.WindowSize) * time.Second

	// Select algorithm based on configuration
	var limiter ratelimiter.RateLimiter
	if s.config.Algorithm == "sliding_window" {
		limiter = s.slidingWindow
	} else {
		limiter = s.leakyBucket
	}

	// Check rate limit
	allowed, err := limiter.Allow(ctx, userID, userLimit, windowSize)
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	return allowed, nil
}

// GetRemaining returns the number of remaining requests for a user
func (s *Service) GetRemaining(ctx context.Context, userID string, limit int) (int, error) {
	userLimit, err := s.getUserLimit(ctx, userID)
	if err != nil {
		userLimit = limit
	}
	if userLimit == 0 {
		userLimit = limit
	}

	windowSize := time.Duration(s.config.WindowSize) * time.Second

	var limiter ratelimiter.RateLimiter
	if s.config.Algorithm == "sliding_window" {
		limiter = s.slidingWindow
	} else {
		limiter = s.leakyBucket
	}

	return limiter.GetRemaining(ctx, userID, userLimit, windowSize)
}

// SetUserLimit sets a custom rate limit for a specific user
// This allows dynamic configuration of rate limits per user
func (s *Service) SetUserLimit(ctx context.Context, userID string, limit int) error {
	key := fmt.Sprintf("rate_limit:config:%s", userID)
	err := s.redisClient.Set(ctx, key, limit, time.Duration(s.config.LocalCacheTTL)*time.Second).Err()
	if err != nil {
		return fmt.Errorf("failed to set user limit: %w", err)
	}

	// Update local cache
	if s.config.EnableLocalCache {
		s.cacheMutex.Lock()
		s.userLimitsCache[userID] = limit
		s.cacheExpiry[userID] = time.Now().Add(time.Duration(s.config.LocalCacheTTL) * time.Second)
		s.cacheMutex.Unlock()
	}

	s.logger.Info("user rate limit updated",
		zap.String("user_id", userID),
		zap.Int("limit", limit),
	)

	return nil
}

// Reset clears the rate limit for a user
func (s *Service) Reset(ctx context.Context, userID string) error {
	var limiter ratelimiter.RateLimiter
	if s.config.Algorithm == "sliding_window" {
		limiter = s.slidingWindow
	} else {
		limiter = s.leakyBucket
	}

	return limiter.Reset(ctx, userID)
}

// getUserLimit retrieves the rate limit for a user
// First checks local cache, then Redis, then returns default
func (s *Service) getUserLimit(ctx context.Context, userID string) (int, error) {
	// Check local cache first
	if s.config.EnableLocalCache {
		s.cacheMutex.RLock()
		if limit, exists := s.userLimitsCache[userID]; exists {
			if expiry, ok := s.cacheExpiry[userID]; ok && time.Now().Before(expiry) {
				s.cacheMutex.RUnlock()
				return limit, nil
			}
		}
		s.cacheMutex.RUnlock()
	}

	// Check Redis
	key := fmt.Sprintf("rate_limit:config:%s", userID)
	val, err := s.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// No custom limit configured, return 0 to use default
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	limit, err := parseInt(val)
	if err != nil {
		return 0, fmt.Errorf("invalid limit value: %w", err)
	}

	// Update local cache
	if s.config.EnableLocalCache {
		s.cacheMutex.Lock()
		s.userLimitsCache[userID] = limit
		s.cacheExpiry[userID] = time.Now().Add(time.Duration(s.config.LocalCacheTTL) * time.Second)
		s.cacheMutex.Unlock()
	}

	return limit, nil
}

// cleanupCache periodically removes expired entries from the local cache
func (s *Service) cleanupCache() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cacheMutex.Lock()
		now := time.Now()
		for userID, expiry := range s.cacheExpiry {
			if now.After(expiry) {
				delete(s.userLimitsCache, userID)
				delete(s.cacheExpiry, userID)
			}
		}
		s.cacheMutex.Unlock()
	}
}

// parseInt safely parses an integer from a string
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	result, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %w", err)
	}
	return result, nil
}
