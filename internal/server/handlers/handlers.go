package handlers

import (
	"demo-saturday/internal/service/ratelimiter"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(
	api *echo.Group,
	rateLimiterService *ratelimiter.Service,
	logger *zap.Logger,
) {
	h := &Handler{
		rateLimiter: rateLimiterService,
		logger:      logger,
	}

	// Test endpoint to demonstrate rate limiting
	api.GET("/test", h.Test)

	// Rate limit management endpoints
	api.POST("/rate-limit/:user_id", h.SetUserLimit)
	api.GET("/rate-limit/:user_id/remaining", h.GetRemaining)
	api.DELETE("/rate-limit/:user_id", h.ResetRateLimit)
}

// Handler contains handler functions
type Handler struct {
	rateLimiter *ratelimiter.Service
	logger      *zap.Logger
}

// Test is a simple endpoint to test rate limiting
func (h *Handler) Test(c echo.Context) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		userID = c.RealIP()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "request successful",
		"user_id": userID,
		"timestamp": c.Request().Header.Get(echo.HeaderXRequestID),
	})
}

// SetUserLimit sets a custom rate limit for a user
func (h *Handler) SetUserLimit(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id is required",
		})
	}

	var req struct {
		Limit int `json:"limit"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if req.Limit <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "limit must be greater than 0",
		})
	}

	if err := h.rateLimiter.SetUserLimit(c.Request().Context(), userID, req.Limit); err != nil {
		h.logger.Error("failed to set user limit",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to set user limit",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "user rate limit updated",
		"user_id": userID,
		"limit":   req.Limit,
	})
}

// GetRemaining returns the remaining requests for a user
func (h *Handler) GetRemaining(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id is required",
		})
	}

	// Get default limit from query parameter or use default
	defaultLimit := 100
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			defaultLimit = limit
		}
	}

	remaining, err := h.rateLimiter.GetRemaining(c.Request().Context(), userID, defaultLimit)
	if err != nil {
		h.logger.Error("failed to get remaining requests",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get remaining requests",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id":   userID,
		"remaining": remaining,
		"limit":     defaultLimit,
	})
}

// ResetRateLimit resets the rate limit for a user
func (h *Handler) ResetRateLimit(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id is required",
		})
	}

	if err := h.rateLimiter.Reset(c.Request().Context(), userID); err != nil {
		h.logger.Error("failed to reset rate limit",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to reset rate limit",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "rate limit reset",
		"user_id": userID,
	})
}

