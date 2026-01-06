package ratelimiter

import (
	"context"
	"time"
)

// RateLimiter defines the interface for rate limiting algorithms
type RateLimiter interface {
	// Allow checks if a request is allowed
	// Returns true if allowed, false if rate limit exceeded
	Allow(ctx context.Context, userID string, limit int, windowSize time.Duration) (bool, error)

	// GetRemaining returns the number of remaining requests allowed
	GetRemaining(ctx context.Context, userID string, limit int, windowSize time.Duration) (int, error)

	// Reset clears the rate limit for a user
	Reset(ctx context.Context, userID string) error
}
