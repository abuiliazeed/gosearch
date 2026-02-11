// Package config provides configuration management and error definitions for gosearch.
//
// It includes error types with helpful suggestions for common issues,
// and configuration loading from files and environment variables.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	// Data directory for index and pages
	DataDir string `mapstructure:"data-dir"`
	// Index path
	IndexPath string `mapstructure:"index-path"`
	// Pages path
	PagesPath string `mapstructure:"pages-path"`
	// Log level
	LogLevel string `mapstructure:"log-level"`
	// Log format (text or json)
	LogFormat string `mapstructure:"log-format"`
	// Verbose output level
	Verbose int `mapstructure:"verbose"`
	// Redis host
	RedisHost string `mapstructure:"redis-host"`
	// Redis password
	RedisPassword string `mapstructure:"redis-password"`
	// Redis cache TTL in seconds
	RedisCacheTTL int `mapstructure:"redis-cache-ttl"`
	// Max crawler workers
	MaxWorkers int `mapstructure:"max-workers"`
	// Crawl delay between requests
	CrawlDelay int `mapstructure:"crawl-delay"`
	// Max crawl depth
	MaxDepth int `mapstructure:"max-depth"`
	// User agent string
	UserAgent string `mapstructure:"user-agent"`
	// Respect robots.txt
	RespectRobots bool `mapstructure:"respect-robots"`
	// Disable cache
	NoCache bool `mapstructure:"no-cache"`
}

// Load loads configuration from file, flags, and environment variables.
// It uses the existing Viper instance from the CLI.
func Load() (*Config, error) {
	config := &Config{}

	// Set defaults
	setDefaults()

	// Bind to struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, NewConfigError(err)
	}

	// Derive paths from data-dir if not explicitly set
	if config.DataDir != "" {
		if config.IndexPath == "" {
			config.IndexPath = filepath.Join(config.DataDir, "index")
		}
		if config.PagesPath == "" {
			config.PagesPath = filepath.Join(config.DataDir, "pages")
		}
	}

	// Validate configuration
	if err := validate(config); err != nil {
		return nil, err
	}

	return config, nil
}

// setDefaults sets default configuration values.
func setDefaults() {
	viper.SetDefault("data-dir", "./data")
	viper.SetDefault("index-path", "")
	viper.SetDefault("pages-path", "")
	viper.SetDefault("log-level", "info")
	viper.SetDefault("log-format", "text")
	viper.SetDefault("verbose", 0)
	viper.SetDefault("redis-host", "localhost:6379")
	viper.SetDefault("redis-password", "")
	viper.SetDefault("redis-cache-ttl", 3600)
	viper.SetDefault("max-workers", 10)
	viper.SetDefault("crawl-delay", 1000)
	viper.SetDefault("max-depth", 3)
	viper.SetDefault("user-agent", "GoSearch/1.0")
	viper.SetDefault("respect-robots", true)
	viper.SetDefault("no-cache", false)
}

// validate validates the configuration.
func validate(cfg *Config) error {
	if cfg.DataDir != "" {
		if err := ensureDir(cfg.DataDir); err != nil {
			return NewConfigError(fmt.Errorf("data directory: %w", err))
		}
	}

	if cfg.IndexPath != "" {
		if err := ensureDir(cfg.IndexPath); err != nil {
			return NewConfigError(fmt.Errorf("index directory: %w", err))
		}
	}

	if cfg.PagesPath != "" {
		if err := ensureDir(cfg.PagesPath); err != nil {
			return NewConfigError(fmt.Errorf("pages directory: %w", err))
		}
	}

	if cfg.MaxWorkers < 1 {
		return NewConfigError(fmt.Errorf("max-workers must be at least 1, got %d", cfg.MaxWorkers))
	}

	if cfg.MaxWorkers > 100 {
		return NewConfigError(fmt.Errorf("max-workers cannot exceed 100, got %d", cfg.MaxWorkers))
	}

	if cfg.CrawlDelay < 0 {
		return NewConfigError(fmt.Errorf("crawl-delay cannot be negative, got %d", cfg.CrawlDelay))
	}

	if cfg.MaxDepth < 0 {
		return NewConfigError(fmt.Errorf("max-depth cannot be negative, got %d", cfg.MaxDepth))
	}

	if cfg.RedisCacheTTL < 0 {
		return NewConfigError(fmt.Errorf("redis-cache-ttl cannot be negative, got %d", cfg.RedisCacheTTL))
	}

	return nil
}

// ensureDir ensures a directory exists, creating it if necessary.
func ensureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}

// GetConfigPath returns the path to the config file if one was loaded.
func GetConfigPath() string {
	if viper.ConfigFileUsed() != "" {
		return viper.ConfigFileUsed()
	}
	return ""
}

// WriteConfig writes the current configuration to a file.
func WriteConfig(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config
	return viper.WriteConfigAs(path)
}

// WriteExampleConfig writes an example configuration file.
func WriteExampleConfig(path string) error {
	// Set example values
	exampleConfig := map[string]interface{}{
		"data-dir":        "./data",
		"log-level":       "info",
		"log-format":      "text",
		"redis-host":      "localhost:6379",
		"redis-cache-ttl": 3600,
		"max-workers":     10,
		"crawl-delay":     1000,
		"max-depth":       3,
		"user-agent":      "GoSearch/1.0",
		"respect-robots":  true,
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write using viper
	v := viper.New()
	for key, value := range exampleConfig {
		v.Set(key, value)
	}

	return v.WriteConfigAs(path)
}
