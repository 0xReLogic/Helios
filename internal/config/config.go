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
}

// ServerConfig holds the server configuration
type ServerConfig struct {
	Port int `yaml:"port"`
}

// BackendConfig holds the backend server configuration
type BackendConfig struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

// LoadBalancerConfig holds the load balancer configuration
type LoadBalancerConfig struct {
	Strategy string `yaml:"strategy"`
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
