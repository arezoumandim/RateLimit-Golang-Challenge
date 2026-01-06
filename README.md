# High-Performance Distributed Rate Limiter

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Redis](https://img.shields.io/badge/Redis-6.0+-DC382D?style=flat&logo=redis)](https://redis.io/)

A high-performance distributed rate limiter for API Gateway that provides request limiting capabilities in distributed environments. Built with Go, Redis, and designed for scalability and precision.

## 📋 Table of Contents

- [Features](#-features)
- [Architecture](#-architecture)
- [Installation and Setup](#-installation-and-setup)
- [Usage](#-usage)
- [Algorithms](#-algorithms)
- [Testing](#-testing)
- [Performance](#-performance)
- [Documentation](#-documentation)
- [License](#-license)

## ✨ Features

- ✅ **Sliding Window Algorithm**: Precise algorithm for high-accuracy rate limiting
- ✅ **Leaky Bucket Algorithm**: Memory-efficient algorithm for uniform traffic
- ✅ **Distributed**: Support for distributed environments using Redis
- ✅ **High Performance**: Optimized for high traffic using Lua scripts and connection pooling
- ✅ **Dynamic Configuration**: Support for different rate limits per user
- ✅ **Local Caching**: Local cache to reduce Redis requests
- ✅ **Thread-Safe**: Uses Go concurrency primitives for safe concurrent operations
- ✅ **Echo Middleware**: Ready-to-use middleware for Echo framework
- ✅ **Docker Support**: Complete Docker and Docker Compose configurations
- ✅ **Comprehensive Tests**: Unit tests and benchmarks included

## 🏗️ Architecture

This project uses Clean Architecture principles:

```
demo-saturday/
├── cmd/                    # Command-line interfaces
│   └── commands/           # Cobra commands
├── internal/               # Internal code (private)
│   ├── app/                # Application layer (Uber FX DI)
│   ├── config/             # Configuration management
│   ├── server/             # HTTP server
│   │   ├── handlers/      # HTTP handlers
│   │   └── middleware/    # Middleware (rate limiter)
│   └── service/            # Business logic
│       └── ratelimiter/    # Rate limiter service
├── pkg/                    # Reusable packages
│   ├── connections/        # Connection managers
│   ├── ratelimiter/        # Rate limiter core implementation
│   └── utility/            # Utilities (logger)
└── tests/                  # Tests
```

For detailed architecture information, see [Architecture Documentation](docs/ARCHITECTURE.md).

## 🚀 Installation and Setup

### Prerequisites

- **Go** 1.21 or higher
- **Redis** 6.0 or higher

### Install Dependencies

```bash
go mod download
go mod tidy
```

### Configuration

Create a `.env` file in the project root:

```env
# Application
APP_ENV=dev
APP_NAME=rate-limiter
APP_VERSION=1.0.0

# API Server
API_PORT=8080
API_HOST=0.0.0.0
API_READ_TIMEOUT=15s
API_WRITE_TIMEOUT=15s
API_IDLE_TIMEOUT=60s
API_SHUTDOWN_TIMEOUT=10s

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Logger
LOGGER_DEVELOPMENT=true
LOGGER_LEVEL=info
LOGGER_ENCODING=json

# Rate Limiter
RATE_LIMIT_DEFAULT_LIMIT=100
RATE_LIMIT_WINDOW_SIZE=1
RATE_LIMIT_ALGORITHM=sliding_window
RATE_LIMIT_ENABLE_LOCAL_CACHE=true
RATE_LIMIT_LOCAL_CACHE_TTL=60
```

For complete environment variable documentation, see [Detailed Guide](docs/DETAILED_GUIDE.md).

### Running

#### Method 1: Direct Execution

```bash
# Start Redis (if not running)
docker run -d -p 6379:6379 redis:7-alpine

# Run the application
go run main.go server
```

#### Method 2: With Docker Compose (Recommended)

```bash
# Production
docker-compose up -d

# Development
docker-compose -f docker-compose.dev.yml up -d

# View logs
docker-compose logs -f app

# Stop
docker-compose down
```

For detailed Docker instructions, see [Docker Guide](docs/DOCKER.md).

#### Method 3: With Docker

```bash
# Build image
docker build -t rate-limiter:latest .

# Run container (requires separate Redis)
docker run -d \
  --name rate-limiter \
  -p 8080:8080 \
  -e REDIS_HOST=host.docker.internal \
  -e REDIS_PORT=6379 \
  rate-limiter:latest
```

## 📖 Usage

### API Endpoints

#### 1. Test Rate Limiting

```bash
curl -H "X-User-ID: user123" http://localhost:8080/api/v1/test
```

#### 2. Set Rate Limit for User

```bash
curl -X POST http://localhost:8080/api/v1/rate-limit/user123 \
  -H "Content-Type: application/json" \
  -d '{"limit": 200}'
```

#### 3. Get Remaining Requests

```bash
curl http://localhost:8080/api/v1/rate-limit/user123/remaining?limit=100
```

#### 4. Reset Rate Limit

```bash
curl -X DELETE http://localhost:8080/api/v1/rate-limit/user123
```

#### 5. Health Check

```bash
curl http://localhost:8080/health
```

### Usage in Code

```go
import (
    "demo-saturday/internal/service/ratelimiter"
    "demo-saturday/pkg/connections"
)

// Create Redis connection
redisClient, err := connections.NewRedis(connections.RedisConfig{
    Host:     "localhost",
    Port:     "6379",
    Password: "",
    DB:       0,
}, logger)

// Create Rate Limiter Service
cfg := &config.RateLimitConfig{
    DefaultLimit:     100,
    WindowSize:       1,
    Algorithm:        "sliding_window",
    EnableLocalCache: true,
    LocalCacheTTL:    60,
}

service := ratelimiter.NewService(redisClient, cfg, logger)

// Check Rate Limit
allowed, err := service.RateLimit(ctx, "user123", 100)
if !allowed {
    // Rate limit exceeded
}
```

### Usage in Echo Middleware

```go
import (
    "demo-saturday/internal/server/middleware"
)

// Add middleware to Echo
e.Use(middleware.RateLimiterMiddleware(
    rateLimiterService,
    logger,
    defaultLimit,
))
```

## 🔄 Algorithms

### Sliding Window

**Advantages:**
- High precision in rate limiting
- Prevents burst traffic
- Uniform distribution of requests

**Trade-offs:**
- Higher memory usage (stores each request)
- Requires cleanup operations

**Usage:**
```go
cfg.Algorithm = "sliding_window"
```

### Leaky Bucket

**Advantages:**
- Lower memory usage
- Simpler implementation
- Suitable for uniform traffic

**Trade-offs:**
- Lower precision compared to Sliding Window
- May cause issues with burst traffic

**Usage:**
```go
cfg.Algorithm = "leaky_bucket"
```

For detailed algorithm explanations and comparisons, see [Detailed Guide](docs/DETAILED_GUIDE.md#rate-limiting-logic).

## 🧪 Testing

### Run Unit Tests

```bash
go test ./tests/ratelimiter/...
```

### Run Tests with Coverage

```bash
go test -cover ./tests/ratelimiter/...
```

### Run Benchmarks

```bash
# Requires running Redis instance
go test -bench=. ./tests/ratelimiter/...
```

### Test Output Example

```
BenchmarkSlidingWindow_Allow-8         10000    150000 ns/op
BenchmarkLeakyBucket_Allow-8            10000    120000 ns/op
BenchmarkService_RateLimit-8            10000    180000 ns/op
BenchmarkService_RateLimit_Concurrent-8  10000    200000 ns/op
```

## ⚡ Performance

### Benchmarks

Benchmark results in test environment (local Redis):

```
BenchmarkSlidingWindow_Allow-8         10000    150000 ns/op
BenchmarkLeakyBucket_Allow-8            10000    120000 ns/op
BenchmarkService_RateLimit-8            10000    180000 ns/op
BenchmarkService_RateLimit_Concurrent-8  10000    200000 ns/op
```

### Optimizations

1. **Lua Scripts**: Atomic operations in Redis
2. **Connection Pooling**: Optimized pool size (50 connections)
3. **Local Caching**: Reduces Redis requests for configurations
4. **Pipeline Operations**: Batch operations for better performance

### Scalability

#### Potential Bottlenecks and Solutions

1. **Redis Performance**
   - Solution: Use Redis Cluster for horizontal scaling
   - Use Redis Sentinel for high availability

2. **Network Latency**
   - Solution: Redis local caching
   - Use Redis replication for read replicas

3. **Memory Usage**
   - Solution: Configure appropriate TTL
   - Use Leaky Bucket to reduce memory usage

#### Scalability Solutions

1. **Horizontal Scaling**: Add more instances
2. **Redis Cluster**: Distribute data across multiple nodes
3. **Caching Strategy**: Local cache for rate limit configs
4. **Monitoring**: Metrics collection and alerting

For more scalability information, see [Architecture Documentation](docs/ARCHITECTURE.md#scalability).

## 📚 Documentation

Comprehensive documentation is available in the `docs/` directory:

- **[Architecture Documentation](docs/ARCHITECTURE.md)** - Complete architecture overview, design decisions, and system design
- **[Detailed Guide](docs/DETAILED_GUIDE.md)** - Comprehensive guide on rate limiting logic, environment variables, usage scenarios, and FAQs
- **[Docker Guide](docs/DOCKER.md)** - Complete Docker setup, usage, and troubleshooting guide
- **[Project Status](docs/PROJECT_STATUS.md)** - Current project status, features, and statistics
- **[Execution Status](docs/EXECUTION_STATUS.md)** - Execution readiness, setup instructions, and troubleshooting

### Quick Links

- [Architecture Overview](docs/ARCHITECTURE.md)
- [Environment Variables Guide](docs/DETAILED_GUIDE.md#environment-variables)
- [Usage Scenarios](docs/DETAILED_GUIDE.md#usage-scenarios)
- [Docker Setup](docs/DOCKER.md)
- [Project Status](docs/PROJECT_STATUS.md)

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is released under the MIT License. See the [LICENSE](LICENSE) file for details.

## 👤 Author

This project was created as a high-performance Rate Limiter implementation example for API Gateway scenarios.

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/)
- Uses [Redis](https://redis.io/) for distributed state management
- HTTP framework: [Echo](https://echo.labstack.com/)
- Dependency Injection: [Uber FX](https://github.com/uber-go/fx)
- Structured Logging: [Zap](https://github.com/uber-go/zap)

---

**⭐ If you find this project useful, please consider giving it a star!**
