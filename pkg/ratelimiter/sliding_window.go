package ratelimiter

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"strconv"
	"time"
)

// SlidingWindow implements a sliding window rate limiter using Redis Sorted Sets
// This algorithm provides high precision and prevents burst traffic exploitation
type SlidingWindow struct {
	client    *redis.Client
	logger    *zap.Logger
	keyPrefix string
}

// NewSlidingWindow creates a new sliding window rate limiter
func NewSlidingWindow(client *redis.Client, logger *zap.Logger) *SlidingWindow {
	return &SlidingWindow{
		client:    client,
		logger:    logger,
		keyPrefix: "rate_limit:sliding:",
	}
}

// Allow checks if a request is allowed based on the sliding window algorithm
// Returns true if allowed, false if rate limit exceeded
//
// Algorithm:
// 1. Use Redis Sorted Set to store request timestamps
// 2. Each request is stored as a member with score = current timestamp
// 3. Remove all entries outside the current window
// 4. Count remaining entries
// 5. If count < limit, allow the request and add current timestamp
//
// This approach ensures:
// - High precision: exact count of requests in the window
// - Fairness: prevents burst traffic from exploiting fixed windows
// - Atomicity: uses Lua script for atomic operations
func (sw *SlidingWindow) Allow(ctx context.Context, userID string, limit int, windowSize time.Duration) (bool, error) {
	key := sw.keyPrefix + userID
	now := time.Now()
	currentTime := now.UnixMilli()
	windowStart := now.Add(-windowSize).UnixMilli()

	// Lua script for atomic operation
	// This ensures all operations happen atomically in Redis
	script := `
		local key = KEYS[1]
		local current_time = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_size_ms = tonumber(ARGV[4])
		
		-- Remove all entries outside the current window
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)
		
		-- Count current requests in the window
		local count = redis.call('ZCARD', key)
		
		-- If under limit, add current request and return 1 (allowed)
		-- Otherwise return 0 (denied)
		if count < limit then
			redis.call('ZADD', key, current_time, current_time)
			-- Set expiration to window size + 1 second for cleanup
			redis.call('EXPIRE', key, math.ceil(window_size_ms / 1000) + 1)
			return 1
		else
			return 0
		end
	`

	result, err := sw.client.Eval(ctx, script, []string{key},
		strconv.FormatInt(currentTime, 10),
		strconv.FormatInt(windowStart, 10),
		strconv.Itoa(limit),
		strconv.FormatInt(windowSize.Milliseconds(), 10),
	).Result()

	if err != nil {
		sw.logger.Error("sliding window rate limit check failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	allowed := result.(int64) == 1

	if !allowed {
		sw.logger.Debug("rate limit exceeded",
			zap.String("user_id", userID),
			zap.Int("limit", limit),
		)
	}

	return allowed, nil
}

// GetRemaining returns the number of remaining requests allowed in the current window
func (sw *SlidingWindow) GetRemaining(ctx context.Context, userID string, limit int, windowSize time.Duration) (int, error) {
	key := sw.keyPrefix + userID
	now := time.Now()
	windowStart := now.Add(-windowSize).UnixMilli()

	// Remove old entries and get count
	pipe := sw.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(windowStart, 10))
	pipe.ZCard(ctx, key)
	results, err := pipe.Exec(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to get remaining requests: %w", err)
	}

	count := results[1].(*redis.IntCmd).Val()
	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// Reset clears the rate limit for a user
func (sw *SlidingWindow) Reset(ctx context.Context, userID string) error {
	key := sw.keyPrefix + userID
	return sw.client.Del(ctx, key).Err()
}
