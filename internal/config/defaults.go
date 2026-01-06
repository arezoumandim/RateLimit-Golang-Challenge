package config

import "github.com/spf13/viper"

func setDefaults() {
	// Application defaults
	viper.SetDefault("app.name", "rate-limiter")
	viper.SetDefault("app.version", "1.0.0")
	viper.SetDefault("app.env", "dev")

	// HTTP server defaults
	viper.SetDefault("api.port", "8080")
	viper.SetDefault("api.host", "0.0.0.0")
	viper.SetDefault("api.read_timeout", "15s")
	viper.SetDefault("api.write_timeout", "15s")
	viper.SetDefault("api.idle_timeout", "60s")
	viper.SetDefault("api.shutdown_timeout", "10s")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Logger defaults
	viper.SetDefault("logger.development", true)
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.encoding", "json")
	viper.SetDefault("logger.log_path", []string{"stdout"})
	viper.SetDefault("logger.error_path", []string{"stderr"})

	// Rate limiter defaults
	viper.SetDefault("rate_limit.default_limit", 100) // 100 requests per second
	viper.SetDefault("rate_limit.window_size", 1)     // 1 second window
	viper.SetDefault("rate_limit.algorithm", "sliding_window")
	viper.SetDefault("rate_limit.enable_local_cache", true)
	viper.SetDefault("rate_limit.local_cache_ttl", 60) // 60 seconds

	// Debug mode
	viper.SetDefault("debug", false)
}
