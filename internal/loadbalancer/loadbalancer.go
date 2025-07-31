package loadbalancer

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

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
	Name         string
	URL          *url.URL
	ReverseProxy *httputil.ReverseProxy
	IsHealthy    bool
}

// LoadBalancer manages the backend servers and implements load balancing
type LoadBalancer struct {
	strategy Strategy
	mutex    sync.RWMutex
}

// NewLoadBalancer creates a new load balancer with the specified strategy
func NewLoadBalancer(cfg *config.Config) (*LoadBalancer, error) {
	var strategy Strategy

	// Create the appropriate strategy based on configuration
	switch cfg.LoadBalancer.Strategy {
	case "round_robin":
		strategy = NewRoundRobinStrategy()
	default:
		// Default to round robin if not specified
		strategy = NewRoundRobinStrategy()
	}

	// Create the load balancer
	lb := &LoadBalancer{
		strategy: strategy,
	}

	// Add backends from configuration
	for _, backendCfg := range cfg.Backends {
		if err := lb.AddBackend(backendCfg); err != nil {
			return nil, err
		}
	}

	return lb, nil
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
		Name:         backendCfg.Name,
		URL:          backendURL,
		ReverseProxy: proxy,
		IsHealthy:    true, // Assume healthy initially
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

// ServeHTTP implements the http.Handler interface
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.NextBackend()
	if backend == nil {
		http.Error(w, "No available backend servers", http.StatusServiceUnavailable)
		return
	}

	// Forward the request to the selected backend
	backend.ReverseProxy.ServeHTTP(w, r)
}
