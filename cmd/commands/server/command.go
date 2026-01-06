package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"demo-saturday/internal/app/server"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewCommand creates a new server command
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start the HTTP server",
		Long:  "Start the HTTP server with rate limiting capabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer()
		},
	}
}

func runServer() error {
	// Create app
	app, err := server.NewApp()
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := app.Start(); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for interrupt signal or error
	select {
	case sig := <-sigChan:
		zap.L().Info("received signal, shutting down gracefully",
			zap.String("signal", sig.String()),
		)
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
		defer shutdownCancel()
		return app.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

