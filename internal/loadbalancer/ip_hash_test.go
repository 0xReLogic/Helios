package loadbalancer

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestIPHashStrategy(t *testing.T) {
	strategy := NewIPHashStrategy()

	backendA := &Backend{Name: "A", URL: &url.URL{}, IsHealthy: true}
	backendB := &Backend{Name: "B", URL: &url.URL{}, IsHealthy: true}
	backendC := &Backend{Name: "C", URL: &url.URL{}, IsHealthy: true}

	strategy.AddBackend(backendA)
	strategy.AddBackend(backendB)
	strategy.AddBackend(backendC)

	// Test with a specific IP
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.168.1.100:12345"

	// The first backend selected for this IP
	expectedBackend := strategy.NextBackend(req1)
	if expectedBackend == nil {
		t.Fatal("Expected a backend, but got nil")
	}

	// Subsequent requests from the same IP should go to the same backend
	for i := 0; i < 10; i++ {
		backend := strategy.NextBackend(req1)
		if backend != expectedBackend {
			t.Errorf("Expected backend %s for IP %s, but got %s", expectedBackend.Name, req1.RemoteAddr, backend.Name)
		}
	}

	// Test with a different IP
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.101:54321"

	expectedBackend2 := strategy.NextBackend(req2)
	if expectedBackend2 == nil {
		t.Fatal("Expected a backend, but got nil")
	}
	for i := 0; i < 10; i++ {
		backend := strategy.NextBackend(req2)
		if backend != expectedBackend2 {
			t.Errorf("Expected backend %s for IP %s, but got %s", expectedBackend2.Name, req2.RemoteAddr, backend.Name)
		}
	}
}

func TestIPHashStrategy_XForwardedFor(t *testing.T) {
	strategy := NewIPHashStrategy()

	backendA := &Backend{Name: "A", URL: &url.URL{}, IsHealthy: true}
	backendB := &Backend{Name: "B", URL: &url.URL{}, IsHealthy: true}

	strategy.AddBackend(backendA)
	strategy.AddBackend(backendB)

	// Create a request with an X-Forwarded-For header
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195")
	req.RemoteAddr = "192.168.1.1:12345" // This should be ignored

	expectedBackend := strategy.NextBackend(req)
	if expectedBackend == nil {
		t.Fatal("Expected a backend, but got nil")
	}

	// Subsequent requests with the same header should go to the same backend
	for i := 0; i < 10; i++ {
		backend := strategy.NextBackend(req)
		if backend != expectedBackend {
			t.Errorf("Expected backend %s for X-Forwarded-For %s, but got %s", expectedBackend.Name, req.Header.Get("X-Forwarded-For"), backend.Name)
		}
	}
}

func TestIPHashStrategy_NoBackends(t *testing.T) {
	strategy := NewIPHashStrategy()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	if strategy.NextBackend(req) != nil {
		t.Error("Expected nil when no backends are available")
	}
}

func TestIPHashStrategy_AllUnhealthy(t *testing.T) {
	strategy := NewIPHashStrategy()

	backendA := &Backend{Name: "A", URL: &url.URL{}, IsHealthy: false}
	backendB := &Backend{Name: "B", URL: &url.URL{}, IsHealthy: false}

	strategy.AddBackend(backendA)
	strategy.AddBackend(backendB)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	if strategy.NextBackend(req) != nil {
		t.Error("Expected nil when all backends are unhealthy")
	}
}
