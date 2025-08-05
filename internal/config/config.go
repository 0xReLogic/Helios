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
}

// ServerConfig holds the server configuration
type ServerConfig struct {
	Port int       `yaml:"port"`
	TLS  TLSConfig `yaml:"tls,omitempty"`
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
	Strategy string `yaml:"strategy"`
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
	if len(config.Backends) == 0 {
		return nil, fmt.Errorf("no backend servers configured")
	}

	return &config, nil
}
