package middleware

import (
	"demo-saturday/internal/service/ratelimiter"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

// RateLimiterMiddleware creates a middleware that enforces rate limiting
// It extracts user ID from the request and checks against the rate limiter
func RateLimiterMiddleware(rateLimiterService *ratelimiter.Service, logger *zap.Logger, defaultLimit int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract user ID from request
			// In a real application, this might come from:
			// - JWT token
			// - API key
			// - Header (X-User-ID)
			// - Query parameter
			// For this example, we'll use X-User-ID header or default to IP address
			userID := c.Request().Header.Get("X-User-ID")
			if userID == "" {
				// Fallback to IP address if no user ID provided
				userID = c.RealIP()
			}

			// Check rate limit
			allowed, err := rateLimiterService.RateLimit(c.Request().Context(), userID, defaultLimit)
			if err != nil {
				logger.Error("rate limit check failed",
					zap.String("user_id", userID),
					zap.Error(err),
				)
				// On error, we allow the request to prevent service degradation
				// In production, you might want to fail closed instead
				return next(c)
			}

			if !allowed {
				// Get remaining requests for better error message
				remaining, _ := rateLimiterService.GetRemaining(c.Request().Context(), userID, defaultLimit)

				logger.Debug("rate limit exceeded",
					zap.String("user_id", userID),
					zap.Int("limit", defaultLimit),
					zap.Int("remaining", remaining),
				)

				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"error":     "rate limit exceeded",
					"message":   "too many requests",
					"retry_after": 1, // seconds
					"remaining": remaining,
				})
			}

			// Get remaining requests and add to response headers
			remaining, _ := rateLimiterService.GetRemaining(c.Request().Context(), userID, defaultLimit)
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(defaultLimit))

			return next(c)
		}
	}
}

