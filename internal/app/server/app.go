package server

import (
	"context"
	"fmt"
	"ratelimit-challenge/internal/config"
	"ratelimit-challenge/internal/server"
	"ratelimit-challenge/internal/service/ratelimiter"
	"ratelimit-challenge/pkg/connections"
	"ratelimit-challenge/pkg/utility"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// App represents the server application
type App struct {
	server *server.Server
	logger *zap.Logger
}

// NewApp creates a new server app with dependency injection
func NewApp() (*App, error) {
	var app *App

	// In Uber FX, fx.Invoke callbacks are executed during fx.New() construction
	// If there's an error in dependency injection, fx.New() will return an error
	fxApp := fx.New(
		fx.Provide(
			config.LoadConfig,
			utility.NewLogger,
			provideRedis,
			ratelimiter.NewService,
			server.NewServer,
		),
		fx.Options(
			fx.NopLogger, // Disable fx default logger
		),
		fx.Invoke(func(
			srv *server.Server,
			logger *zap.Logger,
		) {
			if srv == nil || logger == nil {
				panic(fmt.Sprintf("nil dependencies: srv=%v, logger=%v", srv == nil, logger == nil))
			}
			app = &App{
				server: srv,
				logger: logger,
			}
		}),
	)

	// fx.New() executes Invoke callbacks immediately
	// Start the fx app (this runs lifecycle hooks, not Invoke callbacks)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := fxApp.Start(ctx); err != nil {
		return nil, fmt.Errorf("fx.Start failed: %w", err)
	}

	if app == nil {
		return nil, fmt.Errorf("failed to create app")
	}

	return app, nil
}

// Start starts the server
func (a *App) Start() error {
	return a.server.Start()
}

// Shutdown gracefully shuts down the server
func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

// Provide functions for dependency injection
func provideRedis(cfg *config.Config, logger *zap.Logger) (*redis.Client, error) {
	return connections.NewRedis(connections.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}, logger)
}
