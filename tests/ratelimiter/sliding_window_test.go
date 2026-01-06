package ratelimiter

import (
	"context"
	"ratelimit-challenge/pkg/ratelimiter"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"go.uber.org/zap"
)

// TestSlidingWindow_Allow tests the Allow function using a real Redis instance
// This is an integration test that requires Redis to be running
// To run: go test -tags=integration ./tests/ratelimiter/... -run TestSlidingWindow_Allow
// Or: REDIS_HOST=localhost REDIS_PORT=6379 go test ./tests/ratelimiter/... -run TestSlidingWindow_Allow
func TestSlidingWindow_Allow(t *testing.T) {
	// Skip if Redis is not available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	logger := zap.NewNop()
	sw := ratelimiter.NewSlidingWindow(client, logger)

	userID := "test_user_allow"
	limit := 5
	windowSize := 1 * time.Second

	// Clean up before test
	_ = sw.Reset(ctx, userID)

	t.Run("allow request when under limit", func(t *testing.T) {
		// Make requests up to the limit
		for i := 0; i < limit; i++ {
			allowed, err := sw.Allow(ctx, userID, limit, windowSize)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !allowed {
				t.Errorf("expected request %d to be allowed", i+1)
			}
		}
	})

	t.Run("deny request when over limit", func(t *testing.T) {
		// Use a different user ID to avoid interference
		testUserID := "test_user_deny"
		_ = sw.Reset(ctx, testUserID)

		// Fill up to limit (make exactly 'limit' requests)
		// Add small delay to ensure requests are within the same window
		for i := 0; i < limit; i++ {
			allowed, err := sw.Allow(ctx, testUserID, limit, windowSize)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !allowed {
				t.Errorf("expected request %d to be allowed when filling up", i+1)
			}
			// Small delay to keep requests in the same window
			time.Sleep(10 * time.Millisecond)
		}

		// Next request should be denied (we're at limit)
		allowed, err := sw.Allow(ctx, testUserID, limit, windowSize)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Error("expected request to be denied when over limit")
		}

		// Clean up
		_ = sw.Reset(ctx, testUserID)
	})

	// Clean up after test
	_ = sw.Reset(ctx, userID)
}

func TestSlidingWindow_GetRemaining(t *testing.T) {
	db, mock := redismock.NewClientMock()
	logger := zap.NewNop()
	sw := ratelimiter.NewSlidingWindow(db, logger)

	ctx := context.Background()
	userID := "user123"
	limit := 10
	windowSize := 1 * time.Second

	t.Run("get remaining requests", func(t *testing.T) {
		now := time.Now()
		windowStart := now.Add(-windowSize).UnixMilli()

		// Mock pipeline: remove old entries, get count
		mock.ExpectZRemRangeByScore("rate_limit:sliding:user123", "-inf", strconv.FormatInt(windowStart, 10)).SetVal(0)
		mock.ExpectZCard("rate_limit:sliding:user123").SetVal(3)

		remaining, err := sw.GetRemaining(ctx, userID, limit, windowSize)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := 7 // 10 - 3
		if remaining != expected {
			t.Errorf("expected remaining %d, got %d", expected, remaining)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}

func TestSlidingWindow_Reset(t *testing.T) {
	db, mock := redismock.NewClientMock()
	logger := zap.NewNop()
	sw := ratelimiter.NewSlidingWindow(db, logger)

	ctx := context.Background()
	userID := "user123"

	t.Run("reset rate limit", func(t *testing.T) {
		mock.ExpectDel("rate_limit:sliding:user123").SetVal(1)

		err := sw.Reset(ctx, userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}
