package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load loads the configuration from the specified file
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Read from environment variables
	v.SetEnvPrefix("GATEWAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read from config file
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 5)
	v.SetDefault("server.write_timeout", 10)
	v.SetDefault("server.shutdown_timeout", 5)

	// Proxy defaults
	v.SetDefault("proxy.timeout", 30)
	v.SetDefault("proxy.max_idle_conns", 100)
	v.SetDefault("proxy.idle_conn_timeout", 90)
	v.SetDefault("proxy.enable_cache", false)
	v.SetDefault("proxy.cache_ttl", 60)

	// Security defaults
	v.SetDefault("security.enable_jwt", false)
	v.SetDefault("security.enable_api_key", false)
	v.SetDefault("security.enable_tls", false)
	v.SetDefault("security.enable_cors", true)
	v.SetDefault("security.cors_allow_origins", []string{"*"})

	// Resilience defaults
	v.SetDefault("resilience.enable_circuit_breaker", true)
	v.SetDefault("resilience.failure_threshold", 5)
	v.SetDefault("resilience.reset_timeout", 30)
	v.SetDefault("resilience.enable_retry", true)
	v.SetDefault("resilience.max_retries", 3)
	v.SetDefault("resilience.retry_interval", 100)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output_path", "stdout")

	// Metrics defaults
	v.SetDefault("metrics.enable", true)
	v.SetDefault("metrics.path", "/metrics")
}
