package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure for Helios
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Backends     []BackendConfig    `yaml:"backends"`
	LoadBalancer LoadBalancerConfig `yaml:"load_balancer"`
	HealthChecks HealthChecksConfig `yaml:"health_checks"`
}

// ServerConfig holds the server configuration
type ServerConfig struct {
	Port int `yaml:"port"`
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
