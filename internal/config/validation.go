package config

import (
	"fmt"
)

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	// Validate HTTP config
	if cfg.API.Port == "" {
		return fmt.Errorf("api.port is required")
	}

	// Validate Redis config
	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
	}
	if cfg.Redis.Port == "" {
		return fmt.Errorf("redis.port is required")
	}

	// Validate Rate Limit config
	if cfg.RateLimit.DefaultLimit <= 0 {
		return fmt.Errorf("rate_limit.default_limit must be greater than 0")
	}
	if cfg.RateLimit.WindowSize <= 0 {
		return fmt.Errorf("rate_limit.window_size must be greater than 0")
	}
	if cfg.RateLimit.Algorithm != "sliding_window" && cfg.RateLimit.Algorithm != "leaky_bucket" {
		return fmt.Errorf("rate_limit.algorithm must be either 'sliding_window' or 'leaky_bucket'")
	}

	return nil
}

