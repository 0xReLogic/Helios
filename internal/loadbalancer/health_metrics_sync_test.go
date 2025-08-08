package loadbalancer

import (
	"net/url"
	"testing"
	"time"

	"github.com/0xReLogic/Helios/internal/config"
)

func TestHealthMetricsSynchronization(t *testing.T) {
	cfg := &config.Config{
		LoadBalancer: config.LoadBalancerConfig{Strategy: "round_robin"},
		HealthChecks: config.HealthChecksConfig{
			Active:  config.ActiveHealthCheckConfig{Enabled: false},
			Passive: config.PassiveHealthCheckConfig{Enabled: true, UnhealthyTimeout: 1, UnhealthyThreshold: 1},
		},
		Backends: []config.BackendConfig{
			{Name: "b1", Address: "http://127.0.0.1:65530"},
		},
	}
	lb, err := NewLoadBalancer(cfg)
	if err != nil {
		t.Fatalf("failed to create lb: %v", err)
	}

	// Ensure backend exists
	backs := lb.strategy.GetBackends()
	if len(backs) != 1 {
		t.Fatalf("expected 1 backend, got %d", len(backs))
	}
	b := backs[0]
	if b.URL == nil {
		// sanity
		_, _ = url.Parse("http://127.0.0.1:65530")
	}

	// Initially healthy -> metrics should reflect healthy
	m1 := lb.metricsCollector.GetMetrics()
	bm1, ok := m1.BackendMetrics["b1"]
	if !ok || !bm1.IsHealthy {
		t.Fatalf("expected metrics for b1 healthy=true at init, got: ok=%v healthy=%v", ok, bm1.IsHealthy)
	}

	// Mark unhealthy and check metrics
	lb.MarkBackendUnhealthy(b, 500*time.Millisecond)
	m2 := lb.metricsCollector.GetMetrics()
	bm2, ok := m2.BackendMetrics["b1"]
	if !ok || bm2.IsHealthy {
		t.Fatalf("expected metrics for b1 healthy=false after mark, got: ok=%v healthy=%v", ok, bm2.IsHealthy)
	}
	if bm2.LastHealthCheck.IsZero() {
		t.Fatalf("expected last health check timestamp to be set")
	}

	// Wait for auto-recovery window and trigger check
	time.Sleep(600 * time.Millisecond)
	_ = lb.IsBackendHealthy(b)

	m3 := lb.metricsCollector.GetMetrics()
	bm3, ok := m3.BackendMetrics["b1"]
	if !ok || !bm3.IsHealthy {
		t.Fatalf("expected metrics for b1 healthy=true after recovery, got: ok=%v healthy=%v", ok, bm3.IsHealthy)
	}
}
