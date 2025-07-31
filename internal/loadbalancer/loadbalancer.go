package loadbalancer

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0xReLogic/Helios/internal/config"
)

// Strategy defines the interface for load balancing strategies
type Strategy interface {
	NextBackend() *Backend
	AddBackend(backend *Backend)
	RemoveBackend(backend *Backend)
	GetBackends() []*Backend
}

// Backend represents a backend server
type Backend struct {
	Name              string
	URL               *url.URL
	ReverseProxy      *httputil.ReverseProxy
	IsHealthy         bool
	UnhealthyUntil    time.Time    // Time until which the backend is considered unhealthy
	ActiveConnections int32        // Number of active connections
	Mutex             sync.RWMutex // Mutex for thread-safe operations
}

// healthChecker manages health checks for backends
type healthChecker struct {
	activeEnabled      bool
	activeInterval     time.Duration
	activeTimeout      time.Duration
	activePath         string
	passiveEnabled     bool
	passiveThreshold   int
	passiveTimeout     time.Duration
	unhealthyBackends  map[string]int // Maps backend name to failure count
	unhealthyBackendMu sync.RWMutex
}

// LoadBalancer manages the backend servers and implements load balancing
type LoadBalancer struct {
	strategy     Strategy
	mutex        sync.RWMutex
	config       *config.Config
	healthChecks *healthChecker
}

// NewLoadBalancer creates a new load balancer with the specified strategy
func NewLoadBalancer(cfg *config.Config) (*LoadBalancer, error) {
	var strategy Strategy

	// Create the appropriate strategy based on configuration
	switch cfg.LoadBalancer.Strategy {
	case "round_robin":
		strategy = NewRoundRobinStrategy()
	case "least_connections":
		strategy = NewLeastConnectionsStrategy()
	default:
		// Default to round robin if not specified
		strategy = NewRoundRobinStrategy()
	}

	// Create health checker
	healthChecks := &healthChecker{
		activeEnabled:     cfg.HealthChecks.Active.Enabled,
		activeInterval:    time.Duration(cfg.HealthChecks.Active.Interval) * time.Second,
		activeTimeout:     time.Duration(cfg.HealthChecks.Active.Timeout) * time.Second,
		activePath:        cfg.HealthChecks.Active.Path,
		passiveEnabled:    cfg.HealthChecks.Passive.Enabled,
		passiveThreshold:  cfg.HealthChecks.Passive.UnhealthyThreshold,
		passiveTimeout:    time.Duration(cfg.HealthChecks.Passive.UnhealthyTimeout) * time.Second,
		unhealthyBackends: make(map[string]int),
	}

	// Create the load balancer
	lb := &LoadBalancer{
		strategy:     strategy,
		config:       cfg,
		healthChecks: healthChecks,
	}

	// Add backends from configuration
	for _, backendCfg := range cfg.Backends {
		if err := lb.AddBackend(backendCfg); err != nil {
			return nil, err
		}
	}

	// Start active health checks in a background goroutine if enabled
	if lb.healthChecks.activeEnabled {
		go lb.startActiveHealthChecks()
		log.Printf("Active health checks enabled with interval %v", lb.healthChecks.activeInterval)
	} else {
		log.Printf("Active health checks disabled")
	}

	if lb.healthChecks.passiveEnabled {
		log.Printf("Passive health checks enabled with threshold %d and timeout %v",
			lb.healthChecks.passiveThreshold, lb.healthChecks.passiveTimeout)
	} else {
		log.Printf("Passive health checks disabled")
	}

	return lb, nil
}

// startActiveHealthChecks starts a goroutine that periodically checks the health of all backends
func (lb *LoadBalancer) startActiveHealthChecks() {
	ticker := time.NewTicker(lb.healthChecks.activeInterval)
	defer ticker.Stop()

	log.Printf("Starting active health checks with interval %v", lb.healthChecks.activeInterval)

	// Run an initial health check immediately
	lb.checkBackendsHealth()

	// Use for range instead of for { select {} }
	for range ticker.C {
		lb.checkBackendsHealth()
	}
}

// checkBackendsHealth checks the health of all backends
func (lb *LoadBalancer) checkBackendsHealth() {
	lb.mutex.RLock()
	backends := lb.strategy.GetBackends()
	lb.mutex.RUnlock()

	for _, backend := range backends {
		go lb.checkBackendHealth(backend)
	}
}

// checkBackendHealth checks the health of a single backend
func (lb *LoadBalancer) checkBackendHealth(backend *Backend) {
	// Skip health check if the backend is already marked as unhealthy
	if !lb.IsBackendHealthy(backend) {
		return
	}

	// Create a health check request
	healthURL := *backend.URL
	healthURL.Path = lb.healthChecks.activePath

	req, err := http.NewRequest("GET", healthURL.String(), nil)
	if err != nil {
		log.Printf("Error creating health check request for %s: %v", backend.Name, err)
		return
	}

	// Set a timeout for health checks based on configuration
	client := &http.Client{
		Timeout: lb.healthChecks.activeTimeout,
	}

	// Send the request
	resp, err := client.Do(req)

	// Check for errors or non-200 status codes
	if err != nil {
		log.Printf("Health check failed for %s: %v", backend.Name, err)
		lb.MarkBackendUnhealthy(backend, lb.healthChecks.passiveTimeout)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Health check returned non-OK status for %s: %d", backend.Name, resp.StatusCode)
		lb.MarkBackendUnhealthy(backend, lb.healthChecks.passiveTimeout)
		return
	}

	// If we get here, the backend is healthy
	backend.Mutex.Lock()
	wasUnhealthy := !backend.IsHealthy
	backend.IsHealthy = true
	backend.Mutex.Unlock()

	if wasUnhealthy {
		log.Printf("Backend %s is healthy again (active check)", backend.Name)
	}
}

// AddBackend adds a new backend server to the load balancer
func (lb *LoadBalancer) AddBackend(backendCfg config.BackendConfig) error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Parse the backend URL
	backendURL, err := url.Parse(backendCfg.Address)
	if err != nil {
		return err
	}

	// Create a reverse proxy for this backend
	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Create the backend
	backend := &Backend{
		Name:              backendCfg.Name,
		URL:               backendURL,
		ReverseProxy:      proxy,
		IsHealthy:         true,        // Assume healthy initially
		UnhealthyUntil:    time.Time{}, // Zero time means it's healthy
		ActiveConnections: 0,
	}

	// Add to the strategy
	lb.strategy.AddBackend(backend)

	return nil
}

// RemoveBackend removes a backend server from the load balancer
func (lb *LoadBalancer) RemoveBackend(name string) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Find the backend by name
	for _, backend := range lb.strategy.GetBackends() {
		if backend.Name == name {
			lb.strategy.RemoveBackend(backend)
			break
		}
	}
}

// NextBackend returns the next backend server according to the strategy
func (lb *LoadBalancer) NextBackend() *Backend {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	return lb.strategy.NextBackend()
}

// MarkBackendUnhealthy marks a backend as unhealthy for a specified duration
func (lb *LoadBalancer) MarkBackendUnhealthy(backend *Backend, duration time.Duration) {
	backend.Mutex.Lock()
	defer backend.Mutex.Unlock()

	backend.IsHealthy = false
	backend.UnhealthyUntil = time.Now().Add(duration)

	log.Printf("Backend %s marked as unhealthy for %v", backend.Name, duration)
}

// IsBackendHealthy checks if a backend is currently healthy
func (lb *LoadBalancer) IsBackendHealthy(backend *Backend) bool {
	backend.Mutex.RLock()
	defer backend.Mutex.RUnlock()

	// If it's marked as unhealthy, check if the unhealthy period has expired
	if !backend.IsHealthy {
		if time.Now().After(backend.UnhealthyUntil) {
			// The unhealthy period has expired, mark it as healthy again
			backend.Mutex.RUnlock()
			backend.Mutex.Lock()
			backend.IsHealthy = true
			backend.Mutex.Unlock()
			backend.Mutex.RLock()

			log.Printf("Backend %s is healthy again", backend.Name)
			return true
		}
		return false
	}

	return true
}

// IncrementConnections increments the active connection count for a backend
func (backend *Backend) IncrementConnections() {
	atomic.AddInt32(&backend.ActiveConnections, 1)
}

// DecrementConnections decrements the active connection count for a backend
func (backend *Backend) DecrementConnections() {
	atomic.AddInt32(&backend.ActiveConnections, -1)
}

// GetActiveConnections returns the current number of active connections
func (backend *Backend) GetActiveConnections() int32 {
	return atomic.LoadInt32(&backend.ActiveConnections)
}

// ServeHTTP implements the http.Handler interface
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Find a healthy backend
	var backend *Backend
	for i := 0; i < 3; i++ { // Try up to 3 times to find a healthy backend
		backend = lb.NextBackend()
		if backend == nil {
			http.Error(w, "No available backend servers", http.StatusServiceUnavailable)
			return
		}

		// Check if the backend is healthy
		if !lb.IsBackendHealthy(backend) {
			continue // Try another backend
		}

		// Found a healthy backend
		break
	}

	// If we couldn't find a healthy backend after retries
	if backend == nil || !lb.IsBackendHealthy(backend) {
		http.Error(w, "No healthy backend servers available", http.StatusServiceUnavailable)
		return
	}

	// Track the active connection
	backend.IncrementConnections()

	// Create a custom response writer to capture the status code
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status code
	}

	// Forward the request to the selected backend
	backend.ReverseProxy.ServeHTTP(rw, r)

	// Decrement the connection count when done
	backend.DecrementConnections()

	// Check if the backend returned an error status code (5xx) and passive health checks are enabled
	if rw.statusCode >= 500 && lb.healthChecks.passiveEnabled {
		// Increment failure count for this backend
		lb.healthChecks.unhealthyBackendMu.Lock()
		lb.healthChecks.unhealthyBackends[backend.Name]++
		failureCount := lb.healthChecks.unhealthyBackends[backend.Name]
		lb.healthChecks.unhealthyBackendMu.Unlock()

		log.Printf("Backend %s returned status %d (failure count: %d/%d)",
			backend.Name, rw.statusCode, failureCount, lb.healthChecks.passiveThreshold)

		// If failure count exceeds threshold, mark as unhealthy
		if failureCount >= lb.healthChecks.passiveThreshold {
			lb.MarkBackendUnhealthy(backend, lb.healthChecks.passiveTimeout)

			// Reset failure count
			lb.healthChecks.unhealthyBackendMu.Lock()
			lb.healthChecks.unhealthyBackends[backend.Name] = 0
			lb.healthChecks.unhealthyBackendMu.Unlock()
		}
	}
}

// responseWriter is a custom ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}
