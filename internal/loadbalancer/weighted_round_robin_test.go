package loadbalancer

import (
	"net/url"
	"testing"
)

func TestWeightedRoundRobinStrategy(t *testing.T) {
	strategy := NewWeightedRoundRobinStrategy()

	// Mock backends
	backendA := &Backend{Name: "A", URL: &url.URL{}, Weight: 5, IsHealthy: true}
	backendB := &Backend{Name: "B", URL: &url.URL{}, Weight: 2, IsHealthy: true}
	backendC := &Backend{Name: "C", URL: &url.URL{}, Weight: 1, IsHealthy: true}

	strategy.AddBackend(backendA)
	strategy.AddBackend(backendB)
	strategy.AddBackend(backendC)

	totalWeight := backendA.Weight + backendB.Weight + backendC.Weight // 8
	iterations := totalWeight * 100                                  // 800

	counts := make(map[string]int)
	for i := 0; i < iterations; i++ {
		backend := strategy.NextBackend()
		if backend != nil {
			counts[backend.Name]++
		}
	}

	if len(counts) != 3 {
		t.Errorf("Expected 3 backends to be selected, but got %d", len(counts))
	}

	expectedA := (backendA.Weight * 100)
	expectedB := (backendB.Weight * 100)
	expectedC := (backendC.Weight * 100)

	if counts["A"] != expectedA {
		t.Errorf("Expected backend A to be selected %d times, but got %d", expectedA, counts["A"])
	}
	if counts["B"] != expectedB {
		t.Errorf("Expected backend B to be selected %d times, but got %d", expectedB, counts["B"])
	}
	if counts["C"] != expectedC {
		t.Errorf("Expected backend C to be selected %d times, but got %d", expectedC, counts["C"])
	}
}

func TestWeightedRoundRobinStrategy_WithUnhealthyBackend(t *testing.T) {
	strategy := NewWeightedRoundRobinStrategy()

	backendA := &Backend{Name: "A", URL: &url.URL{}, Weight: 5, IsHealthy: true}
	backendB := &Backend{Name: "B", URL: &url.URL{}, Weight: 2, IsHealthy: false} // B is unhealthy
	backendC := &Backend{Name: "C", URL: &url.URL{}, Weight: 1, IsHealthy: true}

	strategy.AddBackend(backendA)
	strategy.AddBackend(backendB)
	strategy.AddBackend(backendC)

	totalWeight := backendA.Weight + backendC.Weight // 6
	iterations := totalWeight * 100                  // 600

	counts := make(map[string]int)
	for i := 0; i < iterations; i++ {
		backend := strategy.NextBackend()
		if backend != nil {
			counts[backend.Name]++
		}
	}

	if len(counts) != 2 {
		t.Errorf("Expected 2 backends to be selected, but got %d", len(counts))
	}

	if _, exists := counts["B"]; exists {
		t.Errorf("Backend B is unhealthy and should not have been selected, but was selected %d times", counts["B"])
	}

	expectedA := (backendA.Weight * 100)
	expectedC := (backendC.Weight * 100)

	if counts["A"] != expectedA {
		t.Errorf("Expected backend A to be selected %d times, but got %d", expectedA, counts["A"])
	}
	if counts["C"] != expectedC {
		t.Errorf("Expected backend C to be selected %d times, but got %d", expectedC, counts["C"])
	}
}

func TestWeightedRoundRobinStrategy_NoBackends(t *testing.T) {
	strategy := NewWeightedRoundRobinStrategy()
	if strategy.NextBackend() != nil {
		t.Error("Expected nil when no backends are available")
	}
}

func TestWeightedRoundRobinStrategy_AllUnhealthy(t *testing.T) {
	strategy := NewWeightedRoundRobinStrategy()

	backendA := &Backend{Name: "A", URL: &url.URL{}, Weight: 5, IsHealthy: false}
	backendB := &Backend{Name: "B", URL: &url.URL{}, Weight: 2, IsHealthy: false}

	strategy.AddBackend(backendA)
	strategy.AddBackend(backendB)

	if strategy.NextBackend() != nil {
		t.Error("Expected nil when all backends are unhealthy")
	}
}
