package loadbalancer

import (
	"sync"
	"sync/atomic"
)

// RoundRobinStrategy implements a round-robin load balancing strategy
type RoundRobinStrategy struct {
	backends []*Backend
	current  uint64
	mutex    sync.RWMutex
}

// NewRoundRobinStrategy creates a new round-robin strategy
func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{
		backends: make([]*Backend, 0),
		current:  0,
	}
}

// NextBackend returns the next backend in the rotation
func (rr *RoundRobinStrategy) NextBackend() *Backend {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	if len(rr.backends) == 0 {
		return nil
	}

	// Get the next index in a thread-safe way
	idx := atomic.AddUint64(&rr.current, 1) % uint64(len(rr.backends))
	return rr.backends[idx]
}

// AddBackend adds a backend to the pool
func (rr *RoundRobinStrategy) AddBackend(backend *Backend) {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	rr.backends = append(rr.backends, backend)
}

// RemoveBackend removes a backend from the pool
func (rr *RoundRobinStrategy) RemoveBackend(backend *Backend) {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()

	for i, b := range rr.backends {
		if b == backend {
			// Remove the backend by swapping with the last element and truncating
			rr.backends[i] = rr.backends[len(rr.backends)-1]
			rr.backends = rr.backends[:len(rr.backends)-1]
			return
		}
	}
}

// GetBackends returns all backends in the pool
func (rr *RoundRobinStrategy) GetBackends() []*Backend {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	// Return a copy to avoid race conditions
	backends := make([]*Backend, len(rr.backends))
	copy(backends, rr.backends)
	return backends
}
