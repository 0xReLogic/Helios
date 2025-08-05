package loadbalancer

import (
	"math"
	"net/http"
	"sync"
)

// LeastConnectionsStrategy implements a least-connections load balancing strategy
type LeastConnectionsStrategy struct {
	backends []*Backend
	mutex    sync.RWMutex
}

// NewLeastConnectionsStrategy creates a new least-connections strategy
func NewLeastConnectionsStrategy() *LeastConnectionsStrategy {
	return &LeastConnectionsStrategy{
		backends: make([]*Backend, 0),
	}
}

// NextBackend returns the backend with the least active connections
func (lc *LeastConnectionsStrategy) NextBackend(r *http.Request) *Backend {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	if len(lc.backends) == 0 {
		return nil
	}

	var selectedBackend *Backend
	minConnections := int32(math.MaxInt32)

	// Find the backend with the least active connections
	for _, backend := range lc.backends {
		connections := backend.GetActiveConnections()
		if connections < minConnections {
			minConnections = connections
			selectedBackend = backend
		}
	}

	return selectedBackend
}

// AddBackend adds a backend to the pool
func (lc *LeastConnectionsStrategy) AddBackend(backend *Backend) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	lc.backends = append(lc.backends, backend)
}

// RemoveBackend removes a backend from the pool
func (lc *LeastConnectionsStrategy) RemoveBackend(backend *Backend) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	for i, b := range lc.backends {
		if b == backend {
			// Remove the backend by swapping with the last element and truncating
			lc.backends[i] = lc.backends[len(lc.backends)-1]
			lc.backends = lc.backends[:len(lc.backends)-1]
			return
		}
	}
}

// GetBackends returns all backends in the pool
func (lc *LeastConnectionsStrategy) GetBackends() []*Backend {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	// Return a copy to avoid race conditions
	backends := make([]*Backend, len(lc.backends))
	copy(backends, lc.backends)
	return backends
}
