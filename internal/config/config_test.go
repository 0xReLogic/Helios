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

func TestValidateNoBackends(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for no backends, got nil")
	}
}

func TestValidateInvalidServerPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too large", 70000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   ServerConfig{Port: tt.port},
				Backends: []BackendConfig{{Name: "test", Address: "http://localhost:8080"}},
			}
			err := cfg.Validate()
			if err == nil {
				t.Errorf("Expected error for port %d, got nil", tt.port)
			}
		})
	}
}

func TestValidateBackendConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		backend BackendConfig
		wantErr bool
	}{
		{"valid backend", BackendConfig{Name: "test", Address: "http://localhost:8080", Weight: 1}, false},
		{"missing name", BackendConfig{Address: "http://localhost:8080"}, true},
		{"missing address", BackendConfig{Name: "test"}, true},
		{"negative weight", BackendConfig{Name: "test", Address: "http://localhost:8080", Weight: -1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   ServerConfig{Port: 8080},
				Backends: []BackendConfig{tt.backend},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTLSConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		tls     TLSConfig
		wantErr bool
	}{
		{"TLS disabled", TLSConfig{Enabled: false}, false},
		{"TLS with cert and key", TLSConfig{Enabled: true, CertFile: "cert.pem", KeyFile: "key.pem"}, false},
		{"TLS missing cert", TLSConfig{Enabled: true, KeyFile: "key.pem"}, true},
		{"TLS missing key", TLSConfig{Enabled: true, CertFile: "cert.pem"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   ServerConfig{Port: 8080, TLS: tt.tls},
				Backends: []BackendConfig{{Name: "test", Address: "http://localhost:8080"}},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLoadBalancerStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		wantErr  bool
	}{
		{"round_robin", "round_robin", false},
		{"least_connections", "least_connections", false},
		{"weighted_round_robin", "weighted_round_robin", false},
		{"ip_hash", "ip_hash", false},
		{"empty strategy", "", false},
		{"invalid strategy", "random", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:       ServerConfig{Port: 8080},
				Backends:     []BackendConfig{{Name: "test", Address: "http://localhost:8080"}},
				LoadBalancer: LoadBalancerConfig{Strategy: tt.strategy},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateActiveHealthChecks(t *testing.T) {
	tests := []struct {
		name    string
		config  ActiveHealthCheckConfig
		wantErr bool
	}{
		{"disabled", ActiveHealthCheckConfig{Enabled: false}, false},
		{"valid config", ActiveHealthCheckConfig{Enabled: true, Interval: 10, Timeout: 5, Path: "/health"}, false},
		{"zero interval", ActiveHealthCheckConfig{Enabled: true, Interval: 0, Timeout: 5, Path: "/health"}, true},
		{"zero timeout", ActiveHealthCheckConfig{Enabled: true, Interval: 10, Timeout: 0, Path: "/health"}, true},
		{"timeout >= interval", ActiveHealthCheckConfig{Enabled: true, Interval: 5, Timeout: 10, Path: "/health"}, true},
		{"missing path", ActiveHealthCheckConfig{Enabled: true, Interval: 10, Timeout: 5}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   ServerConfig{Port: 8080},
				Backends: []BackendConfig{{Name: "test", Address: "http://localhost:8080"}},
				HealthChecks: HealthChecksConfig{
					Active: tt.config,
				},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassiveHealthChecks(t *testing.T) {
	tests := []struct {
		name    string
		config  PassiveHealthCheckConfig
		wantErr bool
	}{
		{"disabled", PassiveHealthCheckConfig{Enabled: false}, false},
		{"valid config", PassiveHealthCheckConfig{Enabled: true, UnhealthyThreshold: 3, UnhealthyTimeout: 30}, false},
		{"zero threshold", PassiveHealthCheckConfig{Enabled: true, UnhealthyThreshold: 0, UnhealthyTimeout: 30}, true},
		{"zero timeout", PassiveHealthCheckConfig{Enabled: true, UnhealthyThreshold: 3, UnhealthyTimeout: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   ServerConfig{Port: 8080},
				Backends: []BackendConfig{{Name: "test", Address: "http://localhost:8080"}},
				HealthChecks: HealthChecksConfig{
					Passive: tt.config,
				},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRateLimit(t *testing.T) {
	tests := []struct {
		name    string
		config  RateLimitConfig
		wantErr bool
	}{
		{"disabled", RateLimitConfig{Enabled: false}, false},
		{"valid config", RateLimitConfig{Enabled: true, MaxTokens: 100, RefillRate: 1}, false},
		{"zero max tokens", RateLimitConfig{Enabled: true, MaxTokens: 0, RefillRate: 1}, true},
		{"zero refill rate", RateLimitConfig{Enabled: true, MaxTokens: 100, RefillRate: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:    ServerConfig{Port: 8080},
				Backends:  []BackendConfig{{Name: "test", Address: "http://localhost:8080"}},
				RateLimit: tt.config,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCircuitBreaker(t *testing.T) {
	tests := []struct {
		name    string
		config  CircuitBreakerConfig
		wantErr bool
	}{
		{"disabled", CircuitBreakerConfig{Enabled: false}, false},
		{"valid config", CircuitBreakerConfig{Enabled: true, FailureThreshold: 5, SuccessThreshold: 2, TimeoutSeconds: 60, IntervalSeconds: 30}, false},
		{"zero failure threshold", CircuitBreakerConfig{Enabled: true, FailureThreshold: 0, SuccessThreshold: 2, TimeoutSeconds: 60, IntervalSeconds: 30}, true},
		{"zero success threshold", CircuitBreakerConfig{Enabled: true, FailureThreshold: 5, SuccessThreshold: 0, TimeoutSeconds: 60, IntervalSeconds: 30}, true},
		{"zero timeout", CircuitBreakerConfig{Enabled: true, FailureThreshold: 5, SuccessThreshold: 2, TimeoutSeconds: 0, IntervalSeconds: 30}, true},
		{"zero interval", CircuitBreakerConfig{Enabled: true, FailureThreshold: 5, SuccessThreshold: 2, TimeoutSeconds: 60, IntervalSeconds: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:         ServerConfig{Port: 8080},
				Backends:       []BackendConfig{{Name: "test", Address: "http://localhost:8080"}},
				CircuitBreaker: tt.config,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMetrics(t *testing.T) {
	tests := []struct {
		name    string
		config  MetricsConfig
		wantErr bool
	}{
		{"disabled", MetricsConfig{Enabled: false}, false},
		{"valid config", MetricsConfig{Enabled: true, Port: 9090, Path: "/metrics"}, false},
		{"invalid port", MetricsConfig{Enabled: true, Port: 0, Path: "/metrics"}, true},
		{"missing path", MetricsConfig{Enabled: true, Port: 9090}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   ServerConfig{Port: 8080},
				Backends: []BackendConfig{{Name: "test", Address: "http://localhost:8080"}},
				Metrics:  tt.config,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAdminAPI(t *testing.T) {
	tests := []struct {
		name    string
		config  AdminAPIConfig
		wantErr bool
	}{
		{"disabled", AdminAPIConfig{Enabled: false}, false},
		{"valid config", AdminAPIConfig{Enabled: true, Port: 8081}, false},
		{"invalid port", AdminAPIConfig{Enabled: true, Port: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   ServerConfig{Port: 8080},
				Backends: []BackendConfig{{Name: "test", Address: "http://localhost:8080"}},
				AdminAPI: tt.config,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLogging(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggingConfig
		wantErr bool
	}{
		{"valid level and format", LoggingConfig{Level: "info", Format: "json"}, false},
		{"empty level and format", LoggingConfig{}, false},
		{"invalid level", LoggingConfig{Level: "invalid"}, true},
		{"invalid format", LoggingConfig{Format: "invalid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   ServerConfig{Port: 8080},
				Backends: []BackendConfig{{Name: "test", Address: "http://localhost:8080"}},
				Logging:  tt.config,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
