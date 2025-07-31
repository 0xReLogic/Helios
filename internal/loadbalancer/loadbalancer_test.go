package loadbalancer

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"
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

	// Get all backends to verify they're added
	backends := rr.GetBackends()
	if len(backends) != 3 {
		t.Errorf("Expected 3 backends, got %d", len(backends))
	}

	// Test that we can get multiple backends in sequence
	// Note: We don't test the exact order because the implementation might change
	seen := make(map[string]bool)
	for i := 0; i < 3; i++ {
		backend := rr.NextBackend()
		if backend == nil {
			t.Errorf("Expected a backend, got nil")
		} else {
			seen[backend.Name] = true
		}
	}

	// Verify we've seen all backends
	if len(seen) != 3 {
		t.Errorf("Expected to see 3 unique backends, got %d", len(seen))
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
	// Skip this test in automated environments
	if testing.Short() {
		t.Skip("Skipping health check test in short mode")
	}

	// Create a backend directly for testing
	backend := &Backend{
		Name:           "test-backend",
		URL:            &url.URL{Scheme: "http", Host: "localhost:9999"},
		IsHealthy:      true,
		UnhealthyUntil: time.Time{},
	}

	// Create a simple health checker
	healthChecker := &healthChecker{
		activeEnabled:     false,
		passiveEnabled:    true,
		passiveThreshold:  1,
		passiveTimeout:    1 * time.Second,
		unhealthyBackends: make(map[string]int),
	}

	// Create a load balancer manually
	lb := &LoadBalancer{
		strategy:     NewRoundRobinStrategy(),
		healthChecks: healthChecker,
	}

	// Add the backend
	lb.strategy.AddBackend(backend)

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
	// Skip this test in automated environments
	if testing.Short() {
		t.Skip("Skipping ServeHTTP test in short mode")
	}

	// Create a test server that always returns 200 OK
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK from server1"))
	}))
	defer server1.Close()

	// Create a custom round robin strategy that we can control
	strategy := &testStrategy{
		backends: []*Backend{},
		index:    0,
	}

	// Create a backend for the test server
	backendURL, _ := url.Parse(server1.URL)
	backend := &Backend{
		Name:         "test1",
		URL:          backendURL,
		ReverseProxy: httputil.NewSingleHostReverseProxy(backendURL),
		IsHealthy:    true,
	}
	strategy.backends = append(strategy.backends, backend)

	// Create a simple health checker
	healthChecker := &healthChecker{
		activeEnabled:     false,
		passiveEnabled:    false,
		unhealthyBackends: make(map[string]int),
	}

	// Create a load balancer manually
	lb := &LoadBalancer{
		strategy:     strategy,
		healthChecks: healthChecker,
	}

	// Create a test request
	req := httptest.NewRequest("GET", "http://localhost:8080", nil)
	recorder := httptest.NewRecorder()

	// Send the request to the load balancer
	lb.ServeHTTP(recorder, req)

	// Request should go to server1 and return 200 OK
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
}

// testStrategy is a simple strategy for testing
type testStrategy struct {
	backends []*Backend
	index    int
}

func (ts *testStrategy) NextBackend() *Backend {
	if len(ts.backends) == 0 {
		return nil
	}
	backend := ts.backends[ts.index]
	ts.index = (ts.index + 1) % len(ts.backends)
	return backend
}

func (ts *testStrategy) AddBackend(backend *Backend) {
	ts.backends = append(ts.backends, backend)
}

func (ts *testStrategy) RemoveBackend(backend *Backend) {
	for i, b := range ts.backends {
		if b == backend {
			ts.backends = append(ts.backends[:i], ts.backends[i+1:]...)
			return
		}
	}
}

func (ts *testStrategy) GetBackends() []*Backend {
	return ts.backends
}
