package loadbalancer

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/0xReLogic/Helios/internal/config"
)

func TestRoundRobinStrategy(t *testing.T) {
	// Create a round robin strategy
	rr := NewRoundRobinStrategy()

	// Create some test backends
	backend1 := &Backend{
		Name:      "test1",
		URL:       &url.URL{Scheme: "http", Host: "localhost:8081"},
		IsHealthy: true,
	}
	backend2 := &Backend{
		Name:      "test2",
		URL:       &url.URL{Scheme: "http", Host: "localhost:8082"},
		IsHealthy: true,
	}
	backend3 := &Backend{
		Name:      "test3",
		URL:       &url.URL{Scheme: "http", Host: "localhost:8083"},
		IsHealthy: true,
	}

	// Add backends to the strategy
	rr.AddBackend(backend1)
	rr.AddBackend(backend2)
	rr.AddBackend(backend3)

	// Test round robin selection
	selected1 := rr.NextBackend()
	if selected1 != backend1 {
		t.Errorf("Expected backend1, got %s", selected1.Name)
	}

	selected2 := rr.NextBackend()
	if selected2 != backend2 {
		t.Errorf("Expected backend2, got %s", selected2.Name)
	}

	selected3 := rr.NextBackend()
	if selected3 != backend3 {
		t.Errorf("Expected backend3, got %s", selected3.Name)
	}

	// Should wrap around to the first backend
	selected4 := rr.NextBackend()
	if selected4 != backend1 {
		t.Errorf("Expected backend1 again, got %s", selected4.Name)
	}
}

func TestLeastConnectionsStrategy(t *testing.T) {
	// Create a least connections strategy
	lc := NewLeastConnectionsStrategy()

	// Create some test backends with different connection counts
	backend1 := &Backend{
		Name:              "test1",
		URL:               &url.URL{Scheme: "http", Host: "localhost:8081"},
		IsHealthy:         true,
		ActiveConnections: 5,
	}
	backend2 := &Backend{
		Name:              "test2",
		URL:               &url.URL{Scheme: "http", Host: "localhost:8082"},
		IsHealthy:         true,
		ActiveConnections: 2,
	}
	backend3 := &Backend{
		Name:              "test3",
		URL:               &url.URL{Scheme: "http", Host: "localhost:8083"},
		IsHealthy:         true,
		ActiveConnections: 10,
	}

	// Add backends to the strategy
	lc.AddBackend(backend1)
	lc.AddBackend(backend2)
	lc.AddBackend(backend3)

	// Test least connections selection - should select backend2 with fewest connections
	selected := lc.NextBackend()
	if selected != backend2 {
		t.Errorf("Expected backend2 with fewest connections, got %s", selected.Name)
	}

	// Update connection counts
	backend1.ActiveConnections = 1
	backend2.ActiveConnections = 3
	backend3.ActiveConnections = 2

	// Now backend1 should be selected
	selected = lc.NextBackend()
	if selected != backend1 {
		t.Errorf("Expected backend1 with fewest connections, got %s", selected.Name)
	}
}

func TestHealthChecks(t *testing.T) {
	// Create a test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
		Backends: []config.BackendConfig{
			{
				Name:    "test1",
				Address: "http://localhost:8081",
			},
			{
				Name:    "test2",
				Address: "http://localhost:8082",
			},
		},
		LoadBalancer: config.LoadBalancerConfig{
			Strategy: "round_robin",
		},
		HealthChecks: config.HealthChecksConfig{
			Active: config.ActiveHealthCheckConfig{
				Enabled:  true,
				Interval: 1,
				Timeout:  1,
				Path:     "/health",
			},
			Passive: config.PassiveHealthCheckConfig{
				Enabled:            true,
				UnhealthyThreshold: 1,
				UnhealthyTimeout:   5,
			},
		},
	}

	// Create a load balancer
	lb, err := NewLoadBalancer(cfg)
	if err != nil {
		t.Fatalf("Failed to create load balancer: %v", err)
	}

	// Get a backend
	backend := lb.NextBackend()
	if backend == nil {
		t.Fatal("Expected a backend, got nil")
	}

	// Test marking a backend as unhealthy
	lb.MarkBackendUnhealthy(backend, 1*time.Second)

	// Backend should now be unhealthy
	if lb.IsBackendHealthy(backend) {
		t.Error("Expected backend to be unhealthy")
	}

	// Wait for the unhealthy period to expire
	time.Sleep(1100 * time.Millisecond)

	// Backend should now be healthy again
	if !lb.IsBackendHealthy(backend) {
		t.Error("Expected backend to be healthy again")
	}
}

func TestServeHTTP(t *testing.T) {
	// Create a test server that always returns 200 OK
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK from server1"))
	}))
	defer server1.Close()

	// Create a test server that always returns 500 Internal Server Error
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error from server2"))
	}))
	defer server2.Close()

	// Create a test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
		Backends: []config.BackendConfig{
			{
				Name:    "test1",
				Address: server1.URL,
			},
			{
				Name:    "test2",
				Address: server2.URL,
			},
		},
		LoadBalancer: config.LoadBalancerConfig{
			Strategy: "round_robin",
		},
		HealthChecks: config.HealthChecksConfig{
			Active: config.ActiveHealthCheckConfig{
				Enabled:  false,
				Interval: 1,
				Timeout:  1,
				Path:     "/health",
			},
			Passive: config.PassiveHealthCheckConfig{
				Enabled:            true,
				UnhealthyThreshold: 1,
				UnhealthyTimeout:   5,
			},
		},
	}

	// Create a load balancer
	lb, err := NewLoadBalancer(cfg)
	if err != nil {
		t.Fatalf("Failed to create load balancer: %v", err)
	}

	// Create a test request
	req := httptest.NewRequest("GET", "http://localhost:8080", nil)
	recorder := httptest.NewRecorder()

	// Send the request to the load balancer
	lb.ServeHTTP(recorder, req)

	// First request should go to server1 and return 200 OK
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}

	// Send another request
	recorder = httptest.NewRecorder()
	lb.ServeHTTP(recorder, req)

	// Second request should go to server2 and return 500 Internal Server Error
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	// Send a third request
	recorder = httptest.NewRecorder()
	lb.ServeHTTP(recorder, req)

	// Third request should go to server1 again (server2 should be marked as unhealthy)
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
}
