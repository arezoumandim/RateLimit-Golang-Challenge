# Docker Guide

This document is a complete guide on using Docker for the Rate Limiter project.

---

## 📦 Docker Files

### 1. Dockerfile
- **Multi-stage build** for optimizing image size
- **Non-root user** for security
- **Health check** for monitoring
- **Static binary** for portability

### 2. docker-compose.yml
- **Production** configuration
- Includes Redis and Application
- Health checks
- Volume persistence

### 3. docker-compose.dev.yml
- **Development** configuration
- Logging in console mode
- Debug mode enabled

### 4. .dockerignore
- Optimize build context
- Remove unnecessary files

---

## 🚀 Usage

### Method 1: Docker Compose (Recommended)

#### Production
```bash
# Build and run
docker-compose up -d

# View logs
docker-compose logs -f app

# Stop
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

#### Development
```bash
# Build and run
docker-compose -f docker-compose.dev.yml up -d

# View logs
docker-compose -f docker-compose.dev.yml logs -f app

# Stop
docker-compose -f docker-compose.dev.yml down
```

### Method 2: Direct Docker

#### Build Image
```bash
docker build -t rate-limiter:latest .
```

#### Run Container
```bash
# With separate Redis
docker run -d \
  --name rate-limiter \
  -p 8080:8080 \
  -e REDIS_HOST=host.docker.internal \
  -e REDIS_PORT=6379 \
  -e RATE_LIMIT_DEFAULT_LIMIT=100 \
  rate-limiter:latest
```

#### With Redis in Docker
```bash
# 1. Start Redis
docker run -d \
  --name redis \
  -p 6379:6379 \
  redis:7-alpine

# 2. Start Application
docker run -d \
  --name rate-limiter \
  -p 8080:8080 \
  --link redis:redis \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  rate-limiter:latest
```

---

## 🔧 Configuration

### Environment Variables

All settings can be changed through environment variables:

```bash
# Application
APP_ENV=production
APP_NAME=rate-limiter
APP_VERSION=1.0.0

# API Server
API_PORT=8080
API_HOST=0.0.0.0

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Rate Limiter
RATE_LIMIT_DEFAULT_LIMIT=100
RATE_LIMIT_WINDOW_SIZE=1
RATE_LIMIT_ALGORITHM=sliding_window
RATE_LIMIT_ENABLE_LOCAL_CACHE=true
RATE_LIMIT_LOCAL_CACHE_TTL=60
```

### Changing Settings in docker-compose.yml

```yaml
services:
  app:
    environment:
      - RATE_LIMIT_DEFAULT_LIMIT=200  # Change limit
      - RATE_LIMIT_ALGORITHM=leaky_bucket  # Change algorithm
```

---

## 🏥 Health Checks

### Application Health Check
```bash
# Check application health
curl http://localhost:8080/health

# Or in Docker
docker exec rate-limiter-app wget -q -O- http://localhost:8080/health
```

### Redis Health Check
```bash
# Check Redis health
docker exec rate-limiter-redis redis-cli ping
```

---

## 📊 Monitoring

### View Logs
```bash
# All services
docker-compose logs -f

# Application only
docker-compose logs -f app

# Redis only
docker-compose logs -f redis
```

### View Resource Usage
```bash
# Stats
docker stats

# Or for specific service
docker stats rate-limiter-app
```

---

## 🔍 Troubleshooting

### Issue: Container cannot connect to Redis
**Solution**:
```bash
# Check network
docker network ls
docker network inspect rate-limiter-network

# Check Redis
docker exec rate-limiter-redis redis-cli ping
```

### Issue: Port in use
**Solution**:
```bash
# Change port in docker-compose.yml
ports:
  - "8081:8080"  # Use port 8081
```

### Issue: Build fails
**Solution**:
```bash
# Clear cache and rebuild
docker-compose build --no-cache
```

### Issue: Health check fails
**Solution**:
```bash
# Check logs
docker-compose logs app

# Check access to health endpoint
docker exec rate-limiter-app wget -q -O- http://localhost:8080/health
```

---

## 🎯 Best Practices

### 1. Use Multi-stage Build
- Reduce final image size
- More security (no build tools in runtime)

### 2. Non-root User
- Run with non-root user
- Increase security

### 3. Health Checks
- Automatic monitoring
- Automatic restart on failure

### 4. Volume Persistence
- Store Redis data
- Easy backup and restore

### 5. Environment Variables
- Configuration through env vars
- No hardcoding

---

## 📈 Performance

### Image Size
- **Builder stage**: ~300MB
- **Runtime stage**: ~15MB (binary only)
- **Total**: ~15MB (optimized)

### Build Time
- **First build**: ~2-3 minutes
- **Cached build**: ~30 seconds

### Resource Usage
- **Memory**: ~20-30MB
- **CPU**: Minimal (idle)
- **Network**: Low latency with Redis

---

## 🔐 Security

### 1. Non-root User
```dockerfile
USER appuser  # Run with non-root user
```

### 2. Minimal Base Image
```dockerfile
FROM alpine:latest  # Smallest base image
```

### 3. No Secrets in Image
- Use environment variables
- Use secrets management

### 4. Health Checks
- Automatic monitoring
- Early failure detection

---

## 📝 Usage Examples

### Example 1: Development with Hot Reload
```yaml
# docker-compose.dev.yml
services:
  app:
    volumes:
      - .:/app
    command: air  # Requires air for hot reload
```

### Example 2: Production with Custom Config
```yaml
# docker-compose.prod.yml
services:
  app:
    environment:
      - RATE_LIMIT_DEFAULT_LIMIT=1000
      - RATE_LIMIT_ALGORITHM=sliding_window
    deploy:
      replicas: 3
```

### Example 3: With External Redis
```yaml
services:
  app:
    environment:
      - REDIS_HOST=redis.example.com
      - REDIS_PORT=6379
      - REDIS_PASSWORD=secret
```

---

## ✅ Checklist

- [x] Dockerfile with multi-stage build
- [x] docker-compose.yml for production
- [x] docker-compose.dev.yml for development
- [x] .dockerignore for optimization
- [x] Health checks
- [x] Non-root user
- [x] Volume persistence
- [x] Environment variables
- [x] Documentation

---

**Last Updated**: 2024
