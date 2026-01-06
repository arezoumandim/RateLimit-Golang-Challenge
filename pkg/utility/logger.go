package utility

import (
	"ratelimit-challenge/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new zap logger based on configuration
func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	var zapConfig zap.Config

	if cfg.Logger.Development {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	level, err := zapcore.ParseLevel(cfg.Logger.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Set encoding
	if cfg.Logger.Encoding != "" {
		zapConfig.Encoding = cfg.Logger.Encoding
	}

	// Set output paths
	if len(cfg.Logger.OutputPaths) > 0 {
		zapConfig.OutputPaths = cfg.Logger.OutputPaths
	}
	if len(cfg.Logger.ErrorOutputPaths) > 0 {
		zapConfig.ErrorOutputPaths = cfg.Logger.ErrorOutputPaths
	}

	// Build logger
	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}
