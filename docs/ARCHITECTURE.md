# High-Performance Rate Limiter Architecture

## Overview

This project implements a high-performance distributed rate limiter for API Gateway that provides request limiting capabilities in distributed environments.

## Project Structure

```
ratelimit-challenge/
├── cmd/                    # Command-line interfaces
│   └── commands/           # Cobra commands
│       └── server/         # Server command
├── internal/               # Internal code (private)
│   ├── app/                # Application layer
│   │   └── server/         # Server application setup
│   ├── config/             # Configuration management
│   ├── model/              # Data models
│   ├── repository/         # Data access layer
│   ├── server/             # HTTP server
│   │   ├── handlers/       # HTTP handlers
│   │   └── middleware/     # Middleware (rate limiter middleware)
│   └── service/            # Business logic
│       └── ratelimiter/    # Rate limiter service
├── pkg/                    # Reusable packages
│   ├── connections/        # Connection managers
│   │   └── redis.go        # Redis connection
│   └── ratelimiter/        # Rate limiter core implementation
│       ├── sliding_window.go
│       └── leaky_bucket.go
├── docs/                   # Documentation
│   └── ARCHITECTURE.md     # This file
├── tests/                  # Tests
│   └── ratelimiter/        # Rate limiter tests
├── main.go                 # Entry point
└── go.mod                  # Dependencies
```

## Rate Limiter Architecture

### 1. Sliding Window Algorithm

**Advantages:**
- High precision in rate limiting
- Prevents burst traffic
- Uniform distribution of requests over time window

**Implementation:**
- Uses Redis Sorted Sets to store timestamps
- Each request is stored as a member with score = timestamp
- Automatic removal of old entries outside the window
- Counts the number of requests in the current window

**Trade-offs:**
- Higher Redis memory usage (stores each request)
- Requires ZREMRANGEBYSCORE operations for cleanup
- High precision vs. higher memory consumption

### 2. Leaky Bucket Algorithm (Bonus)

**Advantages:**
- Lower memory usage
- Simpler implementation
- Suitable for uniform traffic patterns

**Implementation:**
- Uses Redis INCR and EXPIRE
- Bucket size = limit
- Leak rate = limit per second

**Trade-offs:**
- Lower precision compared to Sliding Window
- May cause issues with burst traffic
- Lower memory usage

### 3. Distributed Architecture

**Challenge:**
- Coordination between multiple service instances
- Ensuring global rate limit compliance

**Solution:**
- Use Redis as single source of truth
- Use Lua scripts for atomic operations
- Reduce race conditions using Redis transactions

### 4. Performance Optimization

**Strategies:**

1. **Connection Pooling:**
   - Use connection pool for Redis
   - Reduce connection overhead

2. **Lua Scripts:**
   - Execute atomic operations in Redis
   - Reduce round-trips to Redis
   - Improve performance under high traffic

3. **Local Caching:**
   - Local cache for rate limit configs
   - Reduce Redis requests for configs

4. **Batch Operations:**
   - Use Pipeline in Redis for multiple operations
   - Reduce latency

5. **Concurrency:**
   - Use Go channels and goroutines
   - Non-blocking operations
   - Context-based cancellation

### 5. Persistence and Recovery

**Strategy:**
- Redis persistence with RDB and AOF
- Recovery from Redis state on restart
- No data loss on crash (with Redis persistence)

**Trade-offs:**
- Redis persistence may reduce performance
- Need backup strategy for production

## Request Flow

```
Client Request
    ↓
Echo Middleware (Rate Limiter)
    ↓
Rate Limiter Service
    ↓
Redis (State Management)
    ↓
Response (Allowed/Denied)
```

## Architectural Decisions

### 1. Using Redis Sorted Sets for Sliding Window

**Reason:**
- Atomic operations for add/remove
- Support for range queries
- Automatic TTL for cleanup

### 2. Lua Scripts for Atomicity

**Reason:**
- Prevents race conditions
- Reduces network round-trips
- Improves performance

### 3. Middleware Pattern

**Reason:**
- Separation of concerns
- Reusability
- Better testability

### 4. Service Layer

**Reason:**
- Separation of business logic from infrastructure
- Unit testability
- Flexibility in changing implementation

## Scalability

### Potential Bottlenecks:

1. **Redis Performance:**
   - Solution: Redis Cluster for horizontal scaling
   - Use Redis Sentinel for high availability

2. **Network Latency:**
   - Solution: Redis local caching
   - Use Redis replication for read replicas

3. **Memory Usage:**
   - Solution: Configure appropriate TTL
   - Use Leaky Bucket to reduce memory usage

### Scalability Solutions:

1. **Horizontal Scaling:**
   - Add more instances
   - Use load balancer

2. **Redis Cluster:**
   - Distribute data across multiple nodes
   - Improve throughput

3. **Caching Strategy:**
   - Local cache for rate limit configs
   - Reduce load on Redis

4. **Monitoring:**
   - Metrics collection
   - Alerting for threshold violations

## Security

1. **User ID Validation:**
   - Prevent injection attacks
   - Input sanitization

2. **Redis Security:**
   - Use password authentication
   - Network isolation

3. **Rate Limit Bypass Prevention:**
   - Use atomic operations
   - Prevent race conditions

## Monitoring and Observability

1. **Metrics:**
   - Number of allowed/denied requests
   - Rate limiter latency
   - Redis connection pool stats

2. **Logging:**
   - Use zap for structured logging
   - Appropriate log level for production

3. **Tracing:**
   - Context propagation
   - Request tracing

## Testability

1. **Unit Tests:**
   - Test rate limiting algorithms
   - Test edge cases

2. **Integration Tests:**
   - Test with Redis
   - Test in distributed environment

3. **Benchmarks:**
   - Performance testing
   - Load testing

## Dependencies

- `github.com/go-redis/redis/v8`: Redis client
- `github.com/labstack/echo/v4`: HTTP framework
- `go.uber.org/zap`: Logging
- `go.uber.org/fx`: Dependency injection
- `github.com/spf13/viper`: Configuration
- `github.com/joho/godotenv`: Environment variables
- `github.com/spf13/cobra`: CLI commands

## Implementation Steps

1. ✅ Create project structure
2. ✅ Implement Redis connection
3. ✅ Implement Sliding Window algorithm
4. ✅ Implement Rate Limiter Service
5. ✅ Create Echo middleware
6. ✅ Implement Leaky Bucket (bonus)
7. ✅ Write Unit Tests
8. ✅ Write Benchmarks
9. ✅ Create README
