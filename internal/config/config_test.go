package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `
server:
  port: 9090

backends:
  - name: "test1"
    address: "http://localhost:9091"
  - name: "test2"
    address: "http://localhost:9092"

load_balancer:
  strategy: "least_connections"

health_checks:
  active:
    enabled: true
    interval: 5
    timeout: 2
    path: "/custom-health"
  passive:
    enabled: true
    unhealthy_threshold: 2
    unhealthy_timeout: 60
`
	tempFile, err := os.CreateTemp("", "helios-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Load the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify server config
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}

	// Verify backends
	if len(cfg.Backends) != 2 {
		t.Errorf("Expected 2 backends, got %d", len(cfg.Backends))
	}
	if cfg.Backends[0].Name != "test1" {
		t.Errorf("Expected backend name 'test1', got '%s'", cfg.Backends[0].Name)
	}
	if cfg.Backends[0].Address != "http://localhost:9091" {
		t.Errorf("Expected backend address 'http://localhost:9091', got '%s'", cfg.Backends[0].Address)
	}
	if cfg.Backends[1].Name != "test2" {
		t.Errorf("Expected backend name 'test2', got '%s'", cfg.Backends[1].Name)
	}
	if cfg.Backends[1].Address != "http://localhost:9092" {
		t.Errorf("Expected backend address 'http://localhost:9092', got '%s'", cfg.Backends[1].Address)
	}

	// Verify load balancer config
	if cfg.LoadBalancer.Strategy != "least_connections" {
		t.Errorf("Expected strategy 'least_connections', got '%s'", cfg.LoadBalancer.Strategy)
	}

	// Verify health checks config
	if !cfg.HealthChecks.Active.Enabled {
		t.Error("Expected active health checks to be enabled")
	}
	if cfg.HealthChecks.Active.Interval != 5 {
		t.Errorf("Expected active health check interval 5, got %d", cfg.HealthChecks.Active.Interval)
	}
	if cfg.HealthChecks.Active.Timeout != 2 {
		t.Errorf("Expected active health check timeout 2, got %d", cfg.HealthChecks.Active.Timeout)
	}
	if cfg.HealthChecks.Active.Path != "/custom-health" {
		t.Errorf("Expected active health check path '/custom-health', got '%s'", cfg.HealthChecks.Active.Path)
	}
	if !cfg.HealthChecks.Passive.Enabled {
		t.Error("Expected passive health checks to be enabled")
	}
	if cfg.HealthChecks.Passive.UnhealthyThreshold != 2 {
		t.Errorf("Expected passive health check threshold 2, got %d", cfg.HealthChecks.Passive.UnhealthyThreshold)
	}
	if cfg.HealthChecks.Passive.UnhealthyTimeout != 60 {
		t.Errorf("Expected passive health check timeout 60, got %d", cfg.HealthChecks.Passive.UnhealthyTimeout)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Create a minimal config file
	configContent := `
server:
  port: 9090

backends:
  - name: "test1"
    address: "http://localhost:9091"
`
	tempFile, err := os.CreateTemp("", "helios-config-minimal-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Load the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Since we're not setting a default in the code, we should just check that it's empty
	// In a real implementation, we would set defaults after loading the config
	if cfg.LoadBalancer.Strategy != "" {
		t.Errorf("Expected empty strategy, got '%s'", cfg.LoadBalancer.Strategy)
	}
}

func TestLoadConfigError(t *testing.T) {
	// Test with non-existent file
	_, err := LoadConfig("non-existent-file.yaml")
	if err == nil {
		t.Error("Expected error when loading non-existent file, got nil")
	}

	// Test with invalid YAML
	tempFile, err := os.CreateTemp("", "helios-config-invalid-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte("invalid: yaml: content:")); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	_, err = LoadConfig(tempFile.Name())
	if err == nil {
		t.Error("Expected error when loading invalid YAML, got nil")
	}
}
