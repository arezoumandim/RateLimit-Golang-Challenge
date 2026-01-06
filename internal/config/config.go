package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

// Config is the root configuration structure
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	API       HTTPConfig      `mapstructure:"api"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Logger    LoggerConfig    `mapstructure:"logger"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Debug     bool            `mapstructure:"debug"`
}

// AppConfig contains application metadata
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"`
}

// HTTPConfig contains HTTP server configuration
type HTTPConfig struct {
	Port            string        `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// RedisConfig contains Redis connection settings
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LoggerConfig contains observability settings
type LoggerConfig struct {
	Development      bool     `mapstructure:"development"`
	Level            string   `mapstructure:"level"`
	Encoding         string   `mapstructure:"encoding"`
	OutputPaths      []string `mapstructure:"log_path"`
	ErrorOutputPaths []string `mapstructure:"error_path"`
}

// RateLimitConfig contains rate limiter configuration
type RateLimitConfig struct {
	// Default rate limit per user (requests per second)
	DefaultLimit int `mapstructure:"default_limit"`
	// Window size in seconds for sliding window
	WindowSize int `mapstructure:"window_size"`
	// Algorithm to use: "sliding_window" or "leaky_bucket"
	Algorithm string `mapstructure:"algorithm"`
	// Enable local caching for rate limit configs
	EnableLocalCache bool `mapstructure:"enable_local_cache"`
	// Local cache TTL in seconds
	LocalCacheTTL int `mapstructure:"local_cache_ttl"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	setDefaults()

	// Initialize Viper
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // Read from system environment variables

	// Load environment files with precedence
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev" // default environment
	}

	// Load .env first (base config)
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	// Load environment-specific .env file (overrides base .env)
	envFile := fmt.Sprintf(".env.%s", env)
	if err := godotenv.Load(envFile); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading %s file: %w", envFile, err)
	}

	// Read from explicit config file if specified (highest precedence)
	if cfgFile := viper.GetString("config"); cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal configuration
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}
