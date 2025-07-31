package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure for Helios
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Backend BackendConfig `yaml:"backend"`
}

// ServerConfig holds the server configuration
type ServerConfig struct {
	Port int `yaml:"port"`
}

// BackendConfig holds the backend server configuration
type BackendConfig struct {
	Address string `yaml:"address"`
}

// LoadConfig loads configuration from the specified YAML file
func LoadConfig(filePath string) (*Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}
