package loadbalancer

import (
	"net/http"
	"sync"
)

// WeightedRoundRobinStrategy implements a smooth weighted round-robin load balancing strategy.
type WeightedRoundRobinStrategy struct {
	backends []*weightedBackend
	mutex    sync.RWMutex
}

// weightedBackend holds the backend and its current weight.
type weightedBackend struct {
	backend       *Backend
	currentWeight int
}

// NewWeightedRoundRobinStrategy creates a new weighted round-robin strategy.
func NewWeightedRoundRobinStrategy() *WeightedRoundRobinStrategy {
	return &WeightedRoundRobinStrategy{
		backends: make([]*weightedBackend, 0),
	}
}

// NextBackend returns the next backend using the smooth weighted round-robin algorithm.
func (wrr *WeightedRoundRobinStrategy) NextBackend(r *http.Request) *Backend {
	wrr.mutex.Lock()
	defer wrr.mutex.Unlock()

	if len(wrr.backends) == 0 {
		return nil
	}

	// This algorithm is based on the smooth weighted round-robin balancing algorithm used in Nginx.
	totalWeight := 0
	var best *weightedBackend

	for _, wb := range wrr.backends {
		// Only consider healthy backends
		if wb.backend.IsHealthy {
			totalWeight += wb.backend.Weight
			wb.currentWeight += wb.backend.Weight

			if best == nil || wb.currentWeight > best.currentWeight {
				best = wb
			}
		}
	}

	if best == nil {
		return nil // All backends are unhealthy
	}

	best.currentWeight -= totalWeight
	return best.backend
}

// AddBackend adds a backend to the pool.
func (wrr *WeightedRoundRobinStrategy) AddBackend(backend *Backend) {
	wrr.mutex.Lock()
	defer wrr.mutex.Unlock()

	weightedBackend := &weightedBackend{
		backend:       backend,
		currentWeight: 0, // Initial weight is 0
	}
	wrr.backends = append(wrr.backends, weightedBackend)
}

// RemoveBackend removes a backend from the pool.
func (wrr *WeightedRoundRobinStrategy) RemoveBackend(backend *Backend) {
	wrr.mutex.Lock()
	defer wrr.mutex.Unlock()

	for i, wb := range wrr.backends {
		if wb.backend == backend {
			// Remove the backend by swapping with the last element and truncating.
			wrr.backends[i] = wrr.backends[len(wrr.backends)-1]
			wrr.backends = wrr.backends[:len(wrr.backends)-1]
			return
		}
	}
}

// GetBackends returns all backends in the pool.
func (wrr *WeightedRoundRobinStrategy) GetBackends() []*Backend {
	wrr.mutex.RLock()
	defer wrr.mutex.RUnlock()

	backends := make([]*Backend, len(wrr.backends))
	for i, wb := range wrr.backends {
		backends[i] = wb.backend
	}
	return backends
}
