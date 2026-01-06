package ratelimiter

import (
	"context"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"ratelimit-challenge/internal/config"
	ratelimiterservice "ratelimit-challenge/internal/service/ratelimiter"
	ratelimiterpkg "ratelimit-challenge/pkg/ratelimiter"
	"testing"
	"time"
)

// BenchmarkSlidingWindow_Allow benchmarks the sliding window rate limiter
func BenchmarkSlidingWindow_Allow(b *testing.B) {
	// Note: This requires a real Redis instance
	// For production benchmarks, use a real Redis connection
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	logger := zap.NewNop()
	sw := ratelimiterpkg.NewSlidingWindow(client, logger)
	userID := "bench_user"
	limit := 1000
	windowSize := 1 * time.Second

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = sw.Allow(ctx, userID, limit, windowSize)
		}
	})
}

// BenchmarkLeakyBucket_Allow benchmarks the leaky bucket rate limiter
func BenchmarkLeakyBucket_Allow(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	logger := zap.NewNop()
	lb := ratelimiterpkg.NewLeakyBucket(client, logger)
	userID := "bench_user"
	limit := 1000
	windowSize := 1 * time.Second

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = lb.Allow(ctx, userID, limit, windowSize)
		}
	})
}

// BenchmarkService_RateLimit benchmarks the rate limiter service
func BenchmarkService_RateLimit(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	cfg := &config.RateLimitConfig{
		DefaultLimit:     1000,
		WindowSize:       1,
		Algorithm:        "sliding_window",
		EnableLocalCache: true,
		LocalCacheTTL:    60,
	}

	logger := zap.NewNop()
	service := ratelimiterservice.NewService(client, cfg, logger)
	ctx := context.Background()
	userID := "bench_user"
	limit := 1000

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = service.RateLimit(ctx, userID, limit)
		}
	})
}

// BenchmarkService_RateLimit_Concurrent benchmarks concurrent rate limiting
func BenchmarkService_RateLimit_Concurrent(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		PoolSize: 50,
	})
	defer client.Close()

	cfg := &config.RateLimitConfig{
		DefaultLimit:     1000,
		WindowSize:       1,
		Algorithm:        "sliding_window",
		EnableLocalCache: true,
		LocalCacheTTL:    60,
	}

	logger := zap.NewNop()
	service := ratelimiterservice.NewService(client, cfg, logger)
	ctx := context.Background()
	limit := 1000

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		userID := "bench_user_concurrent"
		for pb.Next() {
			_, _ = service.RateLimit(ctx, userID, limit)
		}
	})
}
