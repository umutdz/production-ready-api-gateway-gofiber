package config

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Proxy      ProxyConfig      `mapstructure:"proxy"`
	Security   SecurityConfig   `mapstructure:"security"`
	Resilience ResilienceConfig `mapstructure:"resilience"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Metrics    MetricsConfig    `mapstructure:"metrics"`
	Tracing    TracingConfig    `mapstructure:"tracing"`
	Services   []ServiceConfig  `mapstructure:"services"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	ReadTimeout     int    `mapstructure:"read_timeout"`
	WriteTimeout    int    `mapstructure:"write_timeout"`
	ShutdownTimeout int    `mapstructure:"shutdown_timeout"`
	TrustedProxies  []string `mapstructure:"trusted_proxies"`
}

// ProxyConfig contains proxy-related configuration
type ProxyConfig struct {
	Timeout         int  `mapstructure:"timeout"`
	MaxIdleConns    int  `mapstructure:"max_idle_conns"`
	IdleConnTimeout int  `mapstructure:"idle_conn_timeout"`
	EnableCache     bool `mapstructure:"enable_cache"`
	CacheTTL        int  `mapstructure:"cache_ttl"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	EnableJWT       bool   `mapstructure:"enable_jwt"`
	JWTSecret       string `mapstructure:"jwt_secret"`
	EnableAPIKey    bool   `mapstructure:"enable_api_key"`
	APIKeys         []string `mapstructure:"api_keys"`
	EnableTLS       bool   `mapstructure:"enable_tls"`
	TLSCertFile     string `mapstructure:"tls_cert_file"`
	TLSKeyFile      string `mapstructure:"tls_key_file"`
	EnableCORS      bool   `mapstructure:"enable_cors"`
	CORSAllowOrigins []string `mapstructure:"cors_allow_origins"`
}

// ResilienceConfig contains resilience-related configuration
type ResilienceConfig struct {
	EnableCircuitBreaker bool `mapstructure:"enable_circuit_breaker"`
	FailureThreshold     int  `mapstructure:"failure_threshold"`
	ResetTimeout         int  `mapstructure:"reset_timeout"`
	EnableRetry          bool `mapstructure:"enable_retry"`
	MaxRetries           int  `mapstructure:"max_retries"`
	RetryInterval        int  `mapstructure:"retry_interval"`
}

// LoggingConfig contains logging-related configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

// MetricsConfig contains metrics-related configuration
type MetricsConfig struct {
	Enable bool   `mapstructure:"enable"`
	Path   string `mapstructure:"path"`
}

// TracingConfig contains tracing-related configuration
type TracingConfig struct {
	Enable         bool   `mapstructure:"enable"`
	ServiceName    string `mapstructure:"service_name"`
	JaegerEndpoint string `mapstructure:"jaeger_endpoint"`
}

// ServiceConfig contains service-related configuration
type ServiceConfig struct {
	Name           string            `mapstructure:"name"`
	BasePath       string            `mapstructure:"base_path"`
	Targets        []string          `mapstructure:"targets"`
	StripBasePath  bool              `mapstructure:"strip_base_path"`
	EnableWebSocket bool             `mapstructure:"enable_websocket"`
	EnableStickySession bool         `mapstructure:"enable_sticky_session"`
	Headers        map[string]string `mapstructure:"headers"`
	HealthCheck    HealthCheckConfig `mapstructure:"health_check"`
}

// HealthCheckConfig contains health check configuration
type HealthCheckConfig struct {
	Path     string `mapstructure:"path"`
	Interval int    `mapstructure:"interval"`
	Timeout  int    `mapstructure:"timeout"`
}
