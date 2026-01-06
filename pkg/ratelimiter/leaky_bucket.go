package ratelimiter

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"strconv"
	"time"
)

// LeakyBucket implements a leaky bucket rate limiter using Redis
// This algorithm is memory-efficient and suitable for uniform traffic patterns
type LeakyBucket struct {
	client    *redis.Client
	logger    *zap.Logger
	keyPrefix string
}

// NewLeakyBucket creates a new leaky bucket rate limiter
func NewLeakyBucket(client *redis.Client, logger *zap.Logger) *LeakyBucket {
	return &LeakyBucket{
		client:    client,
		logger:    logger,
		keyPrefix: "rate_limit:leaky:",
	}
}

// Allow checks if a request is allowed based on the leaky bucket algorithm
// Returns true if allowed, false if rate limit exceeded
//
// Algorithm:
// 1. Use Redis key to store current bucket level
// 2. Calculate how much has "leaked" since last request
// 3. Update bucket level (subtract leaked amount, add current request)
// 4. If bucket level <= capacity, allow the request
// 5. Otherwise, deny the request
//
// Trade-offs:
// - Lower memory usage (single counter per user)
// - Less precise than sliding window
// - May allow bursts if bucket is empty
func (lb *LeakyBucket) Allow(ctx context.Context, userID string, limit int, windowSize time.Duration) (bool, error) {
	key := lb.keyPrefix + userID
	now := time.Now()
	currentTime := now.UnixMilli()

	// Lua script for atomic operation
	// This ensures bucket level calculation and update happen atomically
	script := `
		local key = KEYS[1]
		local current_time = tonumber(ARGV[1])
		local limit = tonumber(ARGV[2])
		local window_size_ms = tonumber(ARGV[3])
		local leak_rate = limit / (window_size_ms / 1000)  -- requests per millisecond
		
		-- Get current bucket state
		local bucket_data = redis.call('HMGET', key, 'level', 'last_update')
		local level = 0
		local last_update = current_time
		
		if bucket_data[1] then
			level = tonumber(bucket_data[1])
			last_update = tonumber(bucket_data[2])
		end
		
		-- Calculate how much has leaked since last update
		local elapsed = current_time - last_update
		local leaked = elapsed * leak_rate
		
		-- Update bucket level (subtract leaked, ensure non-negative)
		level = math.max(0, level - leaked)
		
		-- Check if we can add the current request
		if level < limit then
			-- Add current request
			level = level + 1
			-- Update bucket state
			redis.call('HMSET', key, 'level', level, 'last_update', current_time)
			-- Set expiration (window size + 1 second)
			redis.call('EXPIRE', key, math.ceil(window_size_ms / 1000) + 1)
			return 1  -- Allowed
		else
			-- Update last_update even if request is denied (for accurate leak calculation)
			redis.call('HSET', key, 'last_update', current_time)
			redis.call('EXPIRE', key, math.ceil(window_size_ms / 1000) + 1)
			return 0  -- Denied
		end
	`

	result, err := lb.client.Eval(ctx, script, []string{key},
		strconv.FormatInt(currentTime, 10),
		strconv.Itoa(limit),
		strconv.FormatInt(windowSize.Milliseconds(), 10),
	).Result()

	if err != nil {
		lb.logger.Error("leaky bucket rate limit check failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	allowed := result.(int64) == 1

	if !allowed {
		lb.logger.Debug("rate limit exceeded (leaky bucket)",
			zap.String("user_id", userID),
			zap.Int("limit", limit),
		)
	}

	return allowed, nil
}

// GetRemaining returns the number of remaining requests allowed in the bucket
func (lb *LeakyBucket) GetRemaining(ctx context.Context, userID string, limit int, windowSize time.Duration) (int, error) {
	key := lb.keyPrefix + userID
	now := time.Now()
	currentTime := now.UnixMilli()

	// Get current bucket state
	bucketData, err := lb.client.HMGet(ctx, key, "level", "last_update").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get bucket state: %w", err)
	}

	// If bucket doesn't exist, full capacity is available
	if bucketData[0] == nil || bucketData[1] == nil {
		return limit, nil
	}

	level, err := strconv.ParseFloat(bucketData[0].(string), 64)
	if err != nil {
		return limit, nil // If parsing fails, assume full capacity
	}
	
	lastUpdate, err := strconv.ParseInt(bucketData[1].(string), 10, 64)
	if err != nil {
		return limit, nil // If parsing fails, assume full capacity
	}

	// Calculate leaked amount
	elapsed := currentTime - lastUpdate
	leakRate := float64(limit) / (float64(windowSize.Milliseconds()) / 1000.0)
	leaked := float64(elapsed) * leakRate

	// Update level
	currentLevel := level - leaked
	if currentLevel < 0 {
		currentLevel = 0
	}

	remaining := limit - int(currentLevel)
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// Reset clears the rate limit for a user
func (lb *LeakyBucket) Reset(ctx context.Context, userID string) error {
	key := lb.keyPrefix + userID
	return lb.client.Del(ctx, key).Err()
}

