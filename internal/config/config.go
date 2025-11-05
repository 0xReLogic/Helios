package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure for Helios
type Config struct {
	Server         ServerConfig         `yaml:"server"`
	Backends       []BackendConfig      `yaml:"backends"`
	LoadBalancer   LoadBalancerConfig   `yaml:"load_balancer"`
	HealthChecks   HealthChecksConfig   `yaml:"health_checks"`
	RateLimit      RateLimitConfig      `yaml:"rate_limit"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	Metrics        MetricsConfig        `yaml:"metrics"`
	AdminAPI       AdminAPIConfig       `yaml:"admin_api"`
	Plugins        PluginsConfig        `yaml:"plugins"`
	Logging        LoggingConfig        `yaml:"logging"`
}

// ServerConfig holds the server configuration
type ServerConfig struct {
	Port     int           `yaml:"port"`
	TLS      TLSConfig     `yaml:"tls,omitempty"`
	Timeouts TimeoutConfig `yaml:"timeouts,omitempty"`
}

// TimeoutConfig holds HTTP server timeout settings
type TimeoutConfig struct {
	Read        int `yaml:"read"`         // ReadTimeout in seconds
	Write       int `yaml:"write"`        // WriteTimeout in seconds
	Idle        int `yaml:"idle"`         // IdleTimeout in seconds
	Handler     int `yaml:"handler"`      // Handler timeout in seconds (end-to-end request)
	Shutdown    int `yaml:"shutdown"`     // Graceful shutdown timeout in seconds
	BackendDial int `yaml:"backend_dial"` // Backend connection dial timeout in seconds
	BackendRead int `yaml:"backend_read"` // Backend response read timeout in seconds
	BackendIdle int `yaml:"backend_idle"` // Backend idle connection timeout in seconds
}

// TLSConfig holds the TLS configuration settings
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
}

// BackendConfig holds the backend server configuration
type BackendConfig struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
	Weight  int    `yaml:"weight,omitempty"`
}

// LoadBalancerConfig holds the load balancer configuration
type LoadBalancerConfig struct {
	Strategy      string              `yaml:"strategy"`
	WebSocketPool WebSocketPoolConfig `yaml:"websocket_pool"`
}

// WebSocketPoolConfig holds WebSocket connection pool settings
type WebSocketPoolConfig struct {
	Enabled            bool `yaml:"enabled"`
	MaxIdle            int  `yaml:"max_idle"`
	MaxActive          int  `yaml:"max_active"`
	IdleTimeoutSeconds int  `yaml:"idle_timeout_seconds"`
}

// HealthChecksConfig holds the health check configuration
type HealthChecksConfig struct {
	Active  ActiveHealthCheckConfig  `yaml:"active"`
	Passive PassiveHealthCheckConfig `yaml:"passive"`
}

// ActiveHealthCheckConfig holds the active health check configuration
type ActiveHealthCheckConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Interval int    `yaml:"interval"`
	Timeout  int    `yaml:"timeout"`
	Path     string `yaml:"path"`
}

// PassiveHealthCheckConfig holds the passive health check configuration
type PassiveHealthCheckConfig struct {
	Enabled            bool `yaml:"enabled"`
	UnhealthyThreshold int  `yaml:"unhealthy_threshold"`
	UnhealthyTimeout   int  `yaml:"unhealthy_timeout"`
}

// RateLimitConfig holds the rate limiting configuration
type RateLimitConfig struct {
	Enabled    bool `yaml:"enabled"`
	MaxTokens  int  `yaml:"max_tokens"`
	RefillRate int  `yaml:"refill_rate_seconds"`
}

// CircuitBreakerConfig holds the circuit breaker configuration
type CircuitBreakerConfig struct {
	Enabled          bool `yaml:"enabled"`
	MaxRequests      int  `yaml:"max_requests"`
	IntervalSeconds  int  `yaml:"interval_seconds"`
	TimeoutSeconds   int  `yaml:"timeout_seconds"`
	FailureThreshold int  `yaml:"failure_threshold"`
	SuccessThreshold int  `yaml:"success_threshold"`
}

// MetricsConfig holds the metrics configuration
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// AdminAPIConfig holds the Admin API configuration
type AdminAPIConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Port      int    `yaml:"port"`
	AuthToken string `yaml:"auth_token,omitempty"`
}

// PluginConfig represents a single plugin in the chain
type PluginConfig struct {
	Name   string                 `yaml:"name"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// PluginsConfig holds plugin system configuration
type PluginsConfig struct {
	Enabled bool           `yaml:"enabled"`
	Chain   []PluginConfig `yaml:"chain"`
}

// LoggingConfig holds the structured logging configuration
type LoggingConfig struct {
	Level         string          `yaml:"level"`
	Format        string          `yaml:"format"`
	IncludeCaller bool            `yaml:"include_caller"`
	RequestID     RequestIDConfig `yaml:"request_id"`
	Trace         TraceConfig     `yaml:"trace"`
}

// RequestIDConfig controls request identifier generation and propagation
type RequestIDConfig struct {
	Enabled bool   `yaml:"enabled"`
	Header  string `yaml:"header"`
}

// TraceConfig controls distributed trace propagation
type TraceConfig struct {
	Enabled bool   `yaml:"enabled"`
	Header  string `yaml:"header"`
}

// LoadConfig loads configuration from the specified YAML file
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() error {
	if err := c.validateBackends(); err != nil {
		return err
	}
	if err := c.validateServer(); err != nil {
		return err
	}
	if err := c.validateTimeouts(); err != nil {
		return err
	}
	if err := c.validateLoadBalancer(); err != nil {
		return err
	}
	if err := c.validateHealthChecks(); err != nil {
		return err
	}
	if err := c.validateRateLimit(); err != nil {
		return err
	}
	if err := c.validateCircuitBreaker(); err != nil {
		return err
	}
	if err := c.validateMetrics(); err != nil {
		return err
	}
	if err := c.validateAdminAPI(); err != nil {
		return err
	}
	if err := c.validateLogging(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateBackends() error {
	if len(c.Backends) == 0 {
		return fmt.Errorf("no backend servers configured")
	}

	for i, backend := range c.Backends {
		if backend.Name == "" {
			return fmt.Errorf("backend %d: name is required", i)
		}
		if backend.Address == "" {
			return fmt.Errorf("backend %s: address is required", backend.Name)
		}
		if backend.Weight < 0 {
			return fmt.Errorf("backend %s: weight must be non-negative (got %d)", backend.Name, backend.Weight)
		}
	}
	return nil
}

func (c *Config) validateServer() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535 (got %d)", c.Server.Port)
	}

	// Validate TLS configuration
	if c.Server.TLS.Enabled {
		if c.Server.TLS.CertFile == "" {
			return fmt.Errorf("TLS enabled but cert file not specified")
		}
		if c.Server.TLS.KeyFile == "" {
			return fmt.Errorf("TLS enabled but key file not specified")
		}
	}
	return nil
}

func (c *Config) validateTimeouts() error {
	if c.Server.Timeouts.Read < 0 {
		return fmt.Errorf("server read timeout must be non-negative (got %d)", c.Server.Timeouts.Read)
	}
	if c.Server.Timeouts.Write < 0 {
		return fmt.Errorf("server write timeout must be non-negative (got %d)", c.Server.Timeouts.Write)
	}
	if c.Server.Timeouts.Idle < 0 {
		return fmt.Errorf("server idle timeout must be non-negative (got %d)", c.Server.Timeouts.Idle)
	}
	if c.Server.Timeouts.Handler < 0 {
		return fmt.Errorf("server handler timeout must be non-negative (got %d)", c.Server.Timeouts.Handler)
	}
	if c.Server.Timeouts.Shutdown < 0 {
		return fmt.Errorf("server shutdown timeout must be non-negative (got %d)", c.Server.Timeouts.Shutdown)
	}
	if c.Server.Timeouts.BackendDial < 0 {
		return fmt.Errorf("backend dial timeout must be non-negative (got %d)", c.Server.Timeouts.BackendDial)
	}
	if c.Server.Timeouts.BackendRead < 0 {
		return fmt.Errorf("backend read timeout must be non-negative (got %d)", c.Server.Timeouts.BackendRead)
	}
	if c.Server.Timeouts.BackendIdle < 0 {
		return fmt.Errorf("backend idle timeout must be non-negative (got %d)", c.Server.Timeouts.BackendIdle)
	}
	return nil
}

func (c *Config) validateLoadBalancer() error {
	// Validate load balancer strategy
	validStrategies := map[string]bool{
		"round_robin":          true,
		"least_connections":    true,
		"weighted_round_robin": true,
		"ip_hash":              true,
	}
	if c.LoadBalancer.Strategy != "" && !validStrategies[c.LoadBalancer.Strategy] {
		return fmt.Errorf("invalid load balancer strategy: %s (valid: round_robin, least_connections, weighted_round_robin, ip_hash)", c.LoadBalancer.Strategy)
	}

	// Validate WebSocket pool configuration if enabled
	if c.LoadBalancer.WebSocketPool.Enabled {
		if c.LoadBalancer.WebSocketPool.MaxIdle < 0 {
			return fmt.Errorf("websocket pool max_idle must be non-negative (got %d)", c.LoadBalancer.WebSocketPool.MaxIdle)
		}
		if c.LoadBalancer.WebSocketPool.MaxActive < 0 {
			return fmt.Errorf("websocket pool max_active must be non-negative (got %d)", c.LoadBalancer.WebSocketPool.MaxActive)
		}
		if c.LoadBalancer.WebSocketPool.MaxActive > 0 && c.LoadBalancer.WebSocketPool.MaxIdle > c.LoadBalancer.WebSocketPool.MaxActive {
			return fmt.Errorf("websocket pool max_idle (%d) must be less than or equal to max_active (%d)", c.LoadBalancer.WebSocketPool.MaxIdle, c.LoadBalancer.WebSocketPool.MaxActive)
		}
		if c.LoadBalancer.WebSocketPool.IdleTimeoutSeconds < 0 {
			return fmt.Errorf("websocket pool idle_timeout_seconds must be non-negative (got %d)", c.LoadBalancer.WebSocketPool.IdleTimeoutSeconds)
		}
	}
	return nil
}

func (c *Config) validateHealthChecks() error {
	// Validate active health checks
	if c.HealthChecks.Active.Enabled {
		if c.HealthChecks.Active.Interval <= 0 {
			return fmt.Errorf("active health check interval must be positive (got %d)", c.HealthChecks.Active.Interval)
		}
		if c.HealthChecks.Active.Timeout <= 0 {
			return fmt.Errorf("active health check timeout must be positive (got %d)", c.HealthChecks.Active.Timeout)
		}
		if c.HealthChecks.Active.Timeout >= c.HealthChecks.Active.Interval {
			return fmt.Errorf("active health check timeout (%d) must be less than interval (%d)", c.HealthChecks.Active.Timeout, c.HealthChecks.Active.Interval)
		}
		if c.HealthChecks.Active.Path == "" {
			return fmt.Errorf("active health check path is required when enabled")
		}
	}

	// Validate passive health checks
	if c.HealthChecks.Passive.Enabled {
		if c.HealthChecks.Passive.UnhealthyThreshold <= 0 {
			return fmt.Errorf("passive health check unhealthy threshold must be positive (got %d)", c.HealthChecks.Passive.UnhealthyThreshold)
		}
		if c.HealthChecks.Passive.UnhealthyTimeout <= 0 {
			return fmt.Errorf("passive health check unhealthy timeout must be positive (got %d)", c.HealthChecks.Passive.UnhealthyTimeout)
		}
	}
	return nil
}

func (c *Config) validateRateLimit() error {
	if c.RateLimit.Enabled {
		if c.RateLimit.MaxTokens <= 0 {
			return fmt.Errorf("rate limit max tokens must be positive (got %d)", c.RateLimit.MaxTokens)
		}
		if c.RateLimit.RefillRate <= 0 {
			return fmt.Errorf("rate limit refill rate must be positive (got %d)", c.RateLimit.RefillRate)
		}
	}
	return nil
}

func (c *Config) validateCircuitBreaker() error {
	if c.CircuitBreaker.Enabled {
		if c.CircuitBreaker.FailureThreshold <= 0 {
			return fmt.Errorf("circuit breaker failure threshold must be positive (got %d)", c.CircuitBreaker.FailureThreshold)
		}
		if c.CircuitBreaker.SuccessThreshold <= 0 {
			return fmt.Errorf("circuit breaker success threshold must be positive (got %d)", c.CircuitBreaker.SuccessThreshold)
		}
		if c.CircuitBreaker.TimeoutSeconds <= 0 {
			return fmt.Errorf("circuit breaker timeout must be positive (got %d)", c.CircuitBreaker.TimeoutSeconds)
		}
		if c.CircuitBreaker.IntervalSeconds <= 0 {
			return fmt.Errorf("circuit breaker interval must be positive (got %d)", c.CircuitBreaker.IntervalSeconds)
		}
	}
	return nil
}

func (c *Config) validateMetrics() error {
	if c.Metrics.Enabled {
		if c.Metrics.Port <= 0 || c.Metrics.Port > 65535 {
			return fmt.Errorf("metrics port must be between 1 and 65535 (got %d)", c.Metrics.Port)
		}
		if c.Metrics.Path == "" {
			return fmt.Errorf("metrics path is required when enabled")
		}
	}
	return nil
}

func (c *Config) validateAdminAPI() error {
	if c.AdminAPI.Enabled {
		if c.AdminAPI.Port <= 0 || c.AdminAPI.Port > 65535 {
			return fmt.Errorf("admin API port must be between 1 and 65535 (got %d)", c.AdminAPI.Port)
		}
	}
	return nil
}

func (c *Config) validateLogging() error {
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}
	if c.Logging.Level != "" && !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error, fatal)", c.Logging.Level)
	}

	validLogFormats := map[string]bool{
		"json":    true,
		"console": true,
	}
	if c.Logging.Format != "" && !validLogFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s (valid: json, console)", c.Logging.Format)
	}
	return nil
}
