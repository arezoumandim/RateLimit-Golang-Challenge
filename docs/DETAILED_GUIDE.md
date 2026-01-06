# Detailed Rate Limiter Guide

This document provides complete and detailed explanations about the project logic, Environment Variables, and their relationships.

---

## 📚 Table of Contents

1. [Rate Limiting Logic](#rate-limiting-logic)
2. [Environment Variables](#environment-variables)
3. [Relationships and Dependencies](#relationships-and-dependencies)
4. [Usage Scenarios](#usage-scenarios)
5. [Practical Examples](#practical-examples)
6. [Frequently Asked Questions](#frequently-asked-questions)

---

## 🧠 Rate Limiting Logic

### 1. General Concept of Rate Limiting

**Rate Limiting** means limiting the number of requests within a specific time period. Main objectives:
- Prevent abuse
- Protect server resources
- Ensure fair usage
- Prevent DDoS attacks

### 2. Sliding Window Algorithm

#### How it Works:
```
Time: 0s -------- 1s -------- 2s -------- 3s
       |          |          |          |
       Request1   Request2   Request3   Request4
       
Window: [----1s----]
        [Request1, Request2]  <- in this window
```

**Practical Example:**
- Limit: 100 requests/second
- Window: 1 second
- At T=0.5s: 50 requests made
- At T=1.0s: window slides forward, requests before T=0.0s are removed
- At T=1.5s: window includes requests between T=0.5s and T=1.5s

#### Implementation in Redis:
```lua
-- 1. Remove old requests (outside window)
ZREMRANGEBYSCORE key -inf window_start

-- 2. Count current requests
count = ZCARD key

-- 3. If count < limit: allow and add
if count < limit then
    ZADD key current_time current_time
    return 1  -- Allowed
else
    return 0  -- Denied
end
```

#### Advantages:
- ✅ High precision: knows exactly the number of requests in the window
- ✅ Prevents burst: cannot use all limit at the beginning of the window
- ✅ Fairness: uniform distribution throughout the window

#### Disadvantages:
- ⚠️ Higher memory consumption: each request is stored as an entry
- ⚠️ Requires cleanup: old requests must be removed

### 3. Leaky Bucket Algorithm

#### How it Works:
```
Bucket with capacity 100:
[████████████░░░░░░░░] 80/100

Every second 10 requests "leak":
[████████░░░░░░░░░░░░] 70/100

New request:
[█████████░░░░░░░░░░░] 80/100  <- added
```

**Practical Example:**
- Limit: 100 requests/second
- Bucket size: 100
- Leak rate: 100 requests/second
- At T=0s: bucket = 0
- At T=0.5s: 50 requests made, bucket = 50
- At T=1.0s: 50 requests leaked, bucket = 0
- At T=1.5s: 30 new requests, bucket = 30

#### Implementation in Redis:
```lua
-- 1. Calculate leak rate
leak_rate = limit / (window_size / 1000)  -- requests per millisecond

-- 2. Calculate elapsed time
elapsed = current_time - last_update

-- 3. Calculate leaked amount
leaked = elapsed * leak_rate

-- 4. Update bucket level
level = max(0, level - leaked)

-- 5. Check if we can add request
if level < limit then
    level = level + 1
    return 1  -- Allowed
else
    return 0  -- Denied
end
```

#### Advantages:
- ✅ Lower memory consumption: only one counter is stored
- ✅ Simplicity: simpler implementation
- ✅ Suitable for uniform traffic

#### Disadvantages:
- ⚠️ Lower precision: may cause issues in burst traffic
- ⚠️ May allow all limit at the beginning of the window

### 4. Algorithm Comparison

| Feature | Sliding Window | Leaky Bucket |
|---------|----------------|--------------|
| Precision | ✅ High | ⚠️ Medium |
| Memory Usage | ⚠️ High | ✅ Low |
| Burst Prevention | ✅ Excellent | ⚠️ Medium |
| Complexity | ⚠️ Medium | ✅ Simple |
| Suitable for | Variable traffic | Uniform traffic |

### 5. Algorithm Selection

**Use Sliding Window when:**
- You need high precision
- Traffic is variable
- You need to prevent bursts
- You have sufficient memory

**Use Leaky Bucket when:**
- Memory consumption is important
- Traffic is uniform
- You need simplicity
- Medium precision is sufficient

---

## 🔧 Environment Variables

### Variable Categories

#### 1. Application Configuration

##### `APP_ENV`
- **Type**: String
- **Default Value**: `dev`
- **Allowed Values**: `dev`, `staging`, `production`
- **Description**: Application execution environment
- **Impact**: 
  - In `dev`: more logging, debug mode
  - In `production`: less logging, performance mode
- **Example**: `APP_ENV=production`

##### `APP_NAME`
- **Type**: String
- **Default Value**: `rate-limiter`
- **Description**: Application name (for logging and monitoring)
- **Example**: `APP_NAME=rate-limiter`

##### `APP_VERSION`
- **Type**: String
- **Default Value**: `1.0.0`
- **Description**: Application version
- **Example**: `APP_VERSION=1.0.0`

#### 2. HTTP Server Configuration

##### `API_PORT`
- **Type**: Integer
- **Default Value**: `8080`
- **Range**: `1-65535`
- **Description**: HTTP server port
- **Example**: `API_PORT=8080`
- **Note**: Make sure the port is available

##### `API_HOST`
- **Type**: String
- **Default Value**: `0.0.0.0`
- **Allowed Values**: `0.0.0.0` (all interfaces), `127.0.0.1` (localhost only)
- **Description**: IP address to bind the server
- **Example**: 
  - `API_HOST=0.0.0.0` (accessible from outside)
  - `API_HOST=127.0.0.1` (localhost only)

##### `API_READ_TIMEOUT`
- **Type**: Duration (string)
- **Default Value**: `15s`
- **Format**: `15s`, `30s`, `1m`
- **Description**: Maximum time to read request
- **Example**: `API_READ_TIMEOUT=15s`
- **Note**: Increase this value for large requests

##### `API_WRITE_TIMEOUT`
- **Type**: Duration (string)
- **Default Value**: `15s`
- **Format**: `15s`, `30s`, `1m`
- **Description**: Maximum time to write response
- **Example**: `API_WRITE_TIMEOUT=15s`

##### `API_IDLE_TIMEOUT`
- **Type**: Duration (string)
- **Default Value**: `60s`
- **Format**: `60s`, `2m`, `5m`
- **Description**: Maximum idle time for connection
- **Example**: `API_IDLE_TIMEOUT=60s`
- **Note**: To reduce the number of open connections

##### `API_SHUTDOWN_TIMEOUT`
- **Type**: Duration (string)
- **Default Value**: `10s`
- **Format**: `10s`, `30s`, `1m`
- **Description**: Maximum time for graceful shutdown
- **Example**: `API_SHUTDOWN_TIMEOUT=10s`

#### 3. Redis Configuration

##### `REDIS_HOST`
- **Type**: String
- **Default Value**: `localhost`
- **Description**: Redis server address
- **Example**: 
  - `REDIS_HOST=localhost` (local)
  - `REDIS_HOST=redis` (Docker network)
  - `REDIS_HOST=redis.example.com` (remote)
- **Note**: In Docker, use the service name

##### `REDIS_PORT`
- **Type**: Integer
- **Default Value**: `6379`
- **Range**: `1-65535`
- **Description**: Redis server port
- **Example**: `REDIS_PORT=6379`

##### `REDIS_PASSWORD`
- **Type**: String
- **Default Value**: `` (empty)
- **Description**: Redis password (if authentication is enabled)
- **Example**: `REDIS_PASSWORD=mysecretpassword`
- **Note**: Must be used in production

##### `REDIS_DB`
- **Type**: Integer
- **Default Value**: `0`
- **Range**: `0-15` (usually)
- **Description**: Database number in Redis
- **Example**: `REDIS_DB=0`
- **Note**: Use for environment separation (dev=0, prod=1)

#### 4. Logger Configuration

##### `LOGGER_DEVELOPMENT`
- **Type**: Boolean
- **Default Value**: `true`
- **Allowed Values**: `true`, `false`
- **Description**: Enable development mode for logging
- **Impact**:
  - `true`: Console encoding, more information
  - `false`: JSON encoding, less information
- **Example**: `LOGGER_DEVELOPMENT=false`

##### `LOGGER_LEVEL`
- **Type**: String
- **Default Value**: `info`
- **Allowed Values**: `debug`, `info`, `warn`, `error`
- **Description**: Logging level
- **Levels**:
  - `debug`: All logs (for development)
  - `info`: General information (default)
  - `warn`: Warnings and errors
  - `error`: Errors only
- **Example**: `LOGGER_LEVEL=info`

##### `LOGGER_ENCODING`
- **Type**: String
- **Default Value**: `json`
- **Allowed Values**: `json`, `console`
- **Description**: Log output format
- **Impact**:
  - `json`: For production and log aggregation
  - `console`: For development and readability
- **Example**: `LOGGER_ENCODING=json`

##### `LOGGER_LOG_PATH`
- **Type**: String (comma-separated)
- **Default Value**: `stdout`
- **Description**: Log output path(s)
- **Example**: 
  - `LOGGER_LOG_PATH=stdout` (console only)
  - `LOGGER_LOG_PATH=stdout,./logs/app.log` (console + file)

##### `LOGGER_ERROR_PATH`
- **Type**: String (comma-separated)
- **Default Value**: `stderr`
- **Description**: Error log output path(s)
- **Example**: `LOGGER_ERROR_PATH=stderr,./logs/error.log`

#### 5. Rate Limiter Configuration

##### `RATE_LIMIT_DEFAULT_LIMIT`
- **Type**: Integer
- **Default Value**: `100`
- **Range**: `1-1000000`
- **Description**: Number of allowed requests per window (by default)
- **Example**: `RATE_LIMIT_DEFAULT_LIMIT=100`
- **Note**: This value is used for users without specific limits

##### `RATE_LIMIT_WINDOW_SIZE`
- **Type**: Integer (seconds)
- **Default Value**: `1`
- **Range**: `1-3600`
- **Description**: Window size in seconds
- **Example**: 
  - `RATE_LIMIT_WINDOW_SIZE=1` (1 second)
  - `RATE_LIMIT_WINDOW_SIZE=60` (1 minute)
- **Note**: Usually 1 second is used

##### `RATE_LIMIT_ALGORITHM`
- **Type**: String
- **Default Value**: `sliding_window`
- **Allowed Values**: `sliding_window`, `leaky_bucket`
- **Description**: Rate limiting algorithm
- **Example**: `RATE_LIMIT_ALGORITHM=sliding_window`
- **Note**: 
  - `sliding_window`: High precision, higher memory consumption
  - `leaky_bucket`: Lower memory consumption, medium precision

##### `RATE_LIMIT_ENABLE_LOCAL_CACHE`
- **Type**: Boolean
- **Default Value**: `true`
- **Allowed Values**: `true`, `false`
- **Description**: Enable local cache for rate limit configs
- **Impact**:
  - `true`: Reduces Redis requests for configs
  - `false`: Always read from Redis
- **Example**: `RATE_LIMIT_ENABLE_LOCAL_CACHE=true`
- **Note**: Usually `true` in production

##### `RATE_LIMIT_LOCAL_CACHE_TTL`
- **Type**: Integer (seconds)
- **Default Value**: `60`
- **Range**: `1-3600`
- **Description**: Local cache validity time (in seconds)
- **Example**: `RATE_LIMIT_LOCAL_CACHE_TTL=60`
- **Note**: 
  - Low value: Faster updates, more requests to Redis
  - High value: Slower updates, fewer requests to Redis

#### 6. Debug Configuration

##### `DEBUG`
- **Type**: Boolean
- **Default Value**: `false`
- **Allowed Values**: `true`, `false`
- **Description**: Enable debug mode
- **Impact**: 
  - `true`: More information in logs
  - `false`: Only essential information
- **Example**: `DEBUG=true`

---

## 🔗 Relationships and Dependencies

### 1. Relationship between Rate Limiter and Redis

```
Application
    │
    ├─> Redis Connection Pool
    │   ├─> Pool Size: 50
    │   ├─> Min Idle: 10
    │   └─> Timeouts: 3-5s
    │
    └─> Rate Limiter Operations
        ├─> Sliding Window: Sorted Sets
        ├─> Leaky Bucket: Hash
        └─> User Configs: Strings
```

**Dependencies:**
- `REDIS_HOST` + `REDIS_PORT`: For connection
- `REDIS_PASSWORD`: For authentication (optional)
- `REDIS_DB`: For data separation

### 2. Relationship between Algorithm and Configuration

```
RATE_LIMIT_ALGORITHM
    │
    ├─> sliding_window
    │   ├─> Uses: Redis Sorted Sets
    │   ├─> Key pattern: rate_limit:sliding:{user_id}
    │   ├─> Memory: Higher (each request is an entry)
    │   └─> Precision: High
    │
    └─> leaky_bucket
        ├─> Uses: Redis Hash
        ├─> Key pattern: rate_limit:leaky:{user_id}
        ├─> Memory: Lower (one counter)
        └─> Precision: Medium
```

### 3. Relationship between Cache and Performance

```
RATE_LIMIT_ENABLE_LOCAL_CACHE
    │
    ├─> true
    │   ├─> RATE_LIMIT_LOCAL_CACHE_TTL: Validity time
    │   ├─> Reduces Redis requests
    │   ├─> Improves Performance
    │   └─> Config may be stale
    │
    └─> false
        ├─> Always read from Redis
        ├─> Instant updates
        └─> More requests to Redis
```

### 4. Relationship between Environment and Configuration

```
APP_ENV
    │
    ├─> dev
    │   ├─> LOGGER_DEVELOPMENT: true
    │   ├─> LOGGER_LEVEL: debug
    │   ├─> LOGGER_ENCODING: console
    │   └─> DEBUG: true
    │
    ├─> staging
    │   ├─> LOGGER_DEVELOPMENT: false
    │   ├─> LOGGER_LEVEL: info
    │   ├─> LOGGER_ENCODING: json
    │   └─> DEBUG: false
    │
    └─> production
        ├─> LOGGER_DEVELOPMENT: false
        ├─> LOGGER_LEVEL: warn
        ├─> LOGGER_ENCODING: json
        └─> DEBUG: false
```

### 5. Relationship between Window Size and Limit

```
RATE_LIMIT_WINDOW_SIZE = 1 second
RATE_LIMIT_DEFAULT_LIMIT = 100

Result: 100 requests per second

If:
RATE_LIMIT_WINDOW_SIZE = 60 seconds
RATE_LIMIT_DEFAULT_LIMIT = 6000

Result: 100 requests per second (same)
```

**Formula:**
```
Rate = LIMIT / WINDOW_SIZE
```

---

## 🎯 Usage Scenarios

### Scenario 1: Development Environment

```env
APP_ENV=dev
API_PORT=8080
API_HOST=0.0.0.0

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

LOGGER_DEVELOPMENT=true
LOGGER_LEVEL=debug
LOGGER_ENCODING=console

RATE_LIMIT_DEFAULT_LIMIT=100
RATE_LIMIT_WINDOW_SIZE=1
RATE_LIMIT_ALGORITHM=sliding_window
RATE_LIMIT_ENABLE_LOCAL_CACHE=true
RATE_LIMIT_LOCAL_CACHE_TTL=60

DEBUG=true
```

**Features:**
- Complete and readable logging
- Debug mode enabled
- Using localhost Redis

### Scenario 2: Production Environment

```env
APP_ENV=production
API_PORT=8080
API_HOST=0.0.0.0

REDIS_HOST=redis-cluster.example.com
REDIS_PORT=6379
REDIS_PASSWORD=secure_password_here
REDIS_DB=1

LOGGER_DEVELOPMENT=false
LOGGER_LEVEL=info
LOGGER_ENCODING=json
LOGGER_LOG_PATH=stdout,/var/log/rate-limiter/app.log
LOGGER_ERROR_PATH=stderr,/var/log/rate-limiter/error.log

RATE_LIMIT_DEFAULT_LIMIT=1000
RATE_LIMIT_WINDOW_SIZE=1
RATE_LIMIT_ALGORITHM=sliding_window
RATE_LIMIT_ENABLE_LOCAL_CACHE=true
RATE_LIMIT_LOCAL_CACHE_TTL=300

DEBUG=false
```

**Features:**
- Logging to JSON (for aggregation)
- Redis with password
- Higher limit (1000)
- Longer cache TTL (5 minutes)

### Scenario 3: High Traffic Environment

```env
RATE_LIMIT_DEFAULT_LIMIT=10000
RATE_LIMIT_WINDOW_SIZE=1
RATE_LIMIT_ALGORITHM=sliding_window
RATE_LIMIT_ENABLE_LOCAL_CACHE=true
RATE_LIMIT_LOCAL_CACHE_TTL=600

REDIS_HOST=redis-cluster
REDIS_PORT=6379
```

**Features:**
- Very high limit
- Longer cache TTL (10 minutes)
- Using Redis Cluster

### Scenario 4: Memory-Constrained Environment

```env
RATE_LIMIT_DEFAULT_LIMIT=100
RATE_LIMIT_WINDOW_SIZE=1
RATE_LIMIT_ALGORITHM=leaky_bucket
RATE_LIMIT_ENABLE_LOCAL_CACHE=true
RATE_LIMIT_LOCAL_CACHE_TTL=60
```

**Features:**
- Using Leaky Bucket (lower memory consumption)
- Medium limit

---

## 💡 Practical Examples

### Example 1: Setup for API with Medium Traffic

```env
# 1000 request per second
RATE_LIMIT_DEFAULT_LIMIT=1000
RATE_LIMIT_WINDOW_SIZE=1
RATE_LIMIT_ALGORITHM=sliding_window
```

**Result:**
- Each user can make 1000 requests per second
- Using Sliding Window for high precision

### Example 2: Setup for API with High Traffic and Limited Memory

```env
# 5000 request per second
RATE_LIMIT_DEFAULT_LIMIT=5000
RATE_LIMIT_WINDOW_SIZE=1
RATE_LIMIT_ALGORITHM=leaky_bucket
```

**Result:**
- Each user can make 5000 requests per second
- Using Leaky Bucket for lower memory consumption

### Example 3: Setup for Rate Limit Based on Minutes

```env
# 6000 request per minute = 100 request per second
RATE_LIMIT_DEFAULT_LIMIT=6000
RATE_LIMIT_WINDOW_SIZE=60
RATE_LIMIT_ALGORITHM=sliding_window
```

**Result:**
- Each user can make 6000 requests per minute
- Window size is 60 seconds

### Example 4: Setting User-Specific Limits

```bash
# Set limit for specific user
curl -X POST http://localhost:8080/api/v1/rate-limit/user123 \
  -H "Content-Type: application/json" \
  -d '{"limit": 500}'
```

**Result:**
- User `user123` can make 500 requests per second
- This value is stored in Redis
- Local cache is valid for 60 seconds (RATE_LIMIT_LOCAL_CACHE_TTL)

---

## ❓ Frequently Asked Questions

### Q1: What is the difference between Sliding Window and Leaky Bucket?

**A:** 
- **Sliding Window**: High precision, higher memory consumption, prevents bursts
- **Leaky Bucket**: Lower memory consumption, medium precision, suitable for uniform traffic

### Q2: When should I use Leaky Bucket?

**A:** When:
- Memory consumption is important
- Traffic is uniform
- Medium precision is sufficient

### Q3: How should I set the Cache TTL?

**A:** 
- **Low (30-60s)**: Faster updates, more requests
- **Medium (60-300s)**: Balance between performance and freshness
- **High (300-600s)**: Better performance, slower updates

### Q4: Why do we use Redis?

**A:**
- **Distributed**: Works across multiple instances
- **Atomic Operations**: Prevents race conditions
- **Performance**: High speed
- **Persistence**: Ability to store data

### Q5: How do I set the limit for a specific user?

**A:**
```bash
curl -X POST http://localhost:8080/api/v1/rate-limit/{user_id} \
  -H "Content-Type: application/json" \
  -d '{"limit": 500}'
```

### Q6: How do I know if rate limit has been exceeded?

**A:**
- Response code: `429 Too Many Requests`
- Response body:
```json
{
  "error": "rate limit exceeded",
  "message": "too many requests",
  "retry_after": 1,
  "remaining": 0
}
```

### Q7: How do I see remaining requests?

**A:**
- Header: `X-RateLimit-Remaining`
- API: `GET /api/v1/rate-limit/{user_id}/remaining`

### Q8: Can I reset the rate limit?

**A:** Yes:
```bash
curl -X DELETE http://localhost:8080/api/v1/rate-limit/{user_id}
```

---

## 📊 Environment Variables Summary Table

| Variable | Type | Default | Range/Values | Critical |
|----------|------|---------|--------------|-----------|
| `APP_ENV` | String | `dev` | dev/staging/production | ✅ |
| `API_PORT` | Integer | `8080` | 1-65535 | ✅ |
| `REDIS_HOST` | String | `localhost` | - | ✅ |
| `REDIS_PORT` | Integer | `6379` | 1-65535 | ✅ |
| `REDIS_PASSWORD` | String | `` | - | ⚠️ |
| `RATE_LIMIT_DEFAULT_LIMIT` | Integer | `100` | 1-1000000 | ✅ |
| `RATE_LIMIT_WINDOW_SIZE` | Integer | `1` | 1-3600 | ✅ |
| `RATE_LIMIT_ALGORITHM` | String | `sliding_window` | sliding_window/leaky_bucket | ✅ |
| `RATE_LIMIT_ENABLE_LOCAL_CACHE` | Boolean | `true` | true/false | ⚠️ |
| `RATE_LIMIT_LOCAL_CACHE_TTL` | Integer | `60` | 1-3600 | ⚠️ |

**Legend:**
- ✅ Critical: Must be configured
- ⚠️ Optional: Optional but recommended

---

**Last Updated**: 2024
