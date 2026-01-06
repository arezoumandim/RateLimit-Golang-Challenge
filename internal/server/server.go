package server

import (
	"context"
	"net/http"
	"ratelimit-challenge/internal/config"
	"ratelimit-challenge/internal/server/handlers"
	ratelimiterMiddleware "ratelimit-challenge/internal/server/middleware"
	"ratelimit-challenge/internal/service/ratelimiter"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	echo        *echo.Echo
	config      *config.Config
	logger      *zap.Logger
	rateLimiter *ratelimiter.Service
}

// NewServer creates a new HTTP server instance
func NewServer(
	cfg *config.Config,
	logger *zap.Logger,
	rateLimiterService *ratelimiter.Service,
) *Server {
	e := echo.New()

	// Hide Echo banner
	e.HideBanner = true

	// Setup middleware
	setupMiddleware(e, logger, cfg, rateLimiterService)

	// Setup routes
	setupRoutes(e, rateLimiterService, logger)

	return &Server{
		echo:        e,
		config:      cfg,
		logger:      logger,
		rateLimiter: rateLimiterService,
	}
}

// setupMiddleware configures Echo middleware
func setupMiddleware(
	e *echo.Echo,
	logger *zap.Logger,
	cfg *config.Config,
	rateLimiterService *ratelimiter.Service,
) {
	// Request ID middleware
	e.Use(echoMiddleware.RequestID())

	// Logger middleware
	e.Use(echoMiddleware.RequestLoggerWithConfig(echoMiddleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		LogMethod:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogValuesFunc: func(c echo.Context, v echoMiddleware.RequestLoggerValues) error {
			logger.Info("request",
				zap.String("id", v.RequestID),
				zap.String("method", v.Method),
				zap.String("uri", v.URI),
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
				zap.String("remote_ip", v.RemoteIP),
				zap.Error(v.Error),
			)
			return nil
		},
	}))

	// Recover middleware
	e.Use(echoMiddleware.Recover())

	// CORS middleware
	e.Use(echoMiddleware.CORS())

	// Health check endpoint (must be before rate limiter to avoid rate limiting)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// Rate limiter middleware (applied to all routes except health check)
	e.Use(ratelimiterMiddleware.RateLimiterMiddleware(
		rateLimiterService,
		logger,
		cfg.RateLimit.DefaultLimit,
	))
}

// setupRoutes configures API routes
func setupRoutes(
	e *echo.Echo,
	rateLimiterService *ratelimiter.Service,
	logger *zap.Logger,
) {

	// API routes
	api := e.Group("/api/v1")
	handlers.RegisterRoutes(api, rateLimiterService, logger)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := s.config.API.Host + ":" + s.config.API.Port
	s.logger.Info("starting HTTP server",
		zap.String("addr", addr),
		zap.String("env", s.config.App.Env),
	)

	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  s.config.API.ReadTimeout,
		WriteTimeout: s.config.API.WriteTimeout,
		IdleTimeout:  s.config.API.IdleTimeout,
	}

	return s.echo.StartServer(server)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")
	return s.echo.Shutdown(ctx)
}
