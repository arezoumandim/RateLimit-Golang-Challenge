package ratelimiter

import (
	"context"
	"ratelimit-challenge/internal/config"
	"ratelimit-challenge/internal/service/ratelimiter"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"go.uber.org/zap"
)

// TestService_RateLimit tests the RateLimit function using a real Redis instance
// This is an integration test that requires Redis to be running
// To run: go test ./tests/ratelimiter/... -run TestService_RateLimit
func TestService_RateLimit(t *testing.T) {
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
	cfg := &config.RateLimitConfig{
		DefaultLimit:     10,
		WindowSize:       1,
		Algorithm:        "sliding_window",
		EnableLocalCache: false,
		LocalCacheTTL:    60,
	}

	service := ratelimiter.NewService(client, cfg, logger)

	userID := "test_user_rate_limit"
	limit := 10

	// Clean up before test
	_ = service.Reset(ctx, userID)

	t.Run("rate limit with default limit", func(t *testing.T) {
		// Make requests up to the limit
		for i := 0; i < limit; i++ {
			allowed, err := service.RateLimit(ctx, userID, limit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !allowed {
				t.Errorf("expected request %d to be allowed", i+1)
			}
		}
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		// Use a different user ID to avoid interference
		testUserID := "test_user_exceeded"
		_ = service.Reset(ctx, testUserID)

		// Fill up to limit (make exactly 'limit' requests)
		// Add small delay to ensure requests are within the same window
		for i := 0; i < limit; i++ {
			allowed, err := service.RateLimit(ctx, testUserID, limit)
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
		allowed, err := service.RateLimit(ctx, testUserID, limit)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Error("expected request to be denied when over limit")
		}

		// Clean up
		_ = service.Reset(ctx, testUserID)
	})

	// Clean up after test
	_ = service.Reset(ctx, userID)
}

func TestService_SetUserLimit(t *testing.T) {
	db, mock := redismock.NewClientMock()
	logger := zap.NewNop()
	cfg := &config.RateLimitConfig{
		DefaultLimit:     10,
		WindowSize:       1,
		Algorithm:        "sliding_window",
		EnableLocalCache: false,
		LocalCacheTTL:    60,
	}

	service := ratelimiter.NewService(db, cfg, logger)
	ctx := context.Background()

	t.Run("set user limit", func(t *testing.T) {
		userID := "user789"
		limit := 50

		mock.ExpectSet("rate_limit:config:user789", limit, 60*time.Second).SetVal("OK")

		err := service.SetUserLimit(ctx, userID, limit)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}

func TestService_GetRemaining(t *testing.T) {
	db, mock := redismock.NewClientMock()
	logger := zap.NewNop()
	cfg := &config.RateLimitConfig{
		DefaultLimit:     10,
		WindowSize:       1,
		Algorithm:        "sliding_window",
		EnableLocalCache: false,
		LocalCacheTTL:    60,
	}

	service := ratelimiter.NewService(db, cfg, logger)
	ctx := context.Background()

	t.Run("get remaining requests", func(t *testing.T) {
		userID := "user999"
		limit := 20

		// Mock: Check for user limit
		mock.ExpectGet("rate_limit:config:user999").RedisNil()

		// Mock: Get remaining (pipeline)
		now := time.Now()
		windowStart := now.Add(-1 * time.Second).UnixMilli()
		mock.ExpectZRemRangeByScore("rate_limit:sliding:user999", "-inf", strconv.FormatInt(windowStart, 10)).SetVal(0)
		mock.ExpectZCard("rate_limit:sliding:user999").SetVal(5)

		remaining, err := service.GetRemaining(ctx, userID, limit)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := 15 // 20 - 5
		if remaining != expected {
			t.Errorf("expected remaining %d, got %d", expected, remaining)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}

func TestService_Reset(t *testing.T) {
	db, mock := redismock.NewClientMock()
	logger := zap.NewNop()
	cfg := &config.RateLimitConfig{
		DefaultLimit:     10,
		WindowSize:       1,
		Algorithm:        "sliding_window",
		EnableLocalCache: false,
		LocalCacheTTL:    60,
	}

	service := ratelimiter.NewService(db, cfg, logger)
	ctx := context.Background()

	t.Run("reset rate limit", func(t *testing.T) {
		userID := "user111"

		mock.ExpectDel("rate_limit:sliding:user111").SetVal(1)

		err := service.Reset(ctx, userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}
