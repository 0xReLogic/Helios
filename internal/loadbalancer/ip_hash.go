package loadbalancer

import (
	"hash/fnv"
	"net"
	"net/http"
	"strings"
	"sync"
)

// IPHashStrategy implements an IP hash load balancing strategy.
type IPHashStrategy struct {
	backends []*Backend
	mutex    sync.RWMutex
}

// NewIPHashStrategy creates a new IP hash strategy.
func NewIPHashStrategy() *IPHashStrategy {
	return &IPHashStrategy{
		backends: make([]*Backend, 0),
	}
}

// NextBackend returns the next backend using the IP hash algorithm.
func (iph *IPHashStrategy) NextBackend(r *http.Request) *Backend {
	iph.mutex.RLock()
	defer iph.mutex.RUnlock()

	if len(iph.backends) == 0 {
		return nil
	}

	// Get healthy backends
	healthyBackends := make([]*Backend, 0)
	for _, b := range iph.backends {
		if b.IsHealthy {
			healthyBackends = append(healthyBackends, b)
		}
	}

	if len(healthyBackends) == 0 {
		return nil
	}

	// Get the client's IP address
	// In a real-world scenario, you might want to trust X-Forwarded-For or X-Real-IP
	// based on your infrastructure setup.
	ipStr := r.Header.Get("X-Forwarded-For")
	if ipStr == "" {
		ipStr = r.Header.Get("X-Real-IP")
	}
	if ipStr == "" {
		// Fallback to RemoteAddr
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// If SplitHostPort fails, it might be because there is no port.
			ipStr = r.RemoteAddr
		} else {
			ipStr = ip
		}
	}

	// If X-Forwarded-For has multiple IPs, take the first one.
	if strings.Contains(ipStr, ",") {
		ipStr = strings.Split(ipStr, ",")[0]
	}

	// Hash the IP address
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(ipStr)) // #nosec G104 - hash.Write never returns an error for fnv

	hashValue := hash.Sum32()

	// Select a backend
	index := int(hashValue % uint32(len(healthyBackends))) // #nosec G115 - len() is always non-negative, safe conversion
	return healthyBackends[index]
}

// AddBackend adds a backend to the pool.
func (iph *IPHashStrategy) AddBackend(backend *Backend) {
	iph.mutex.Lock()
	defer iph.mutex.Unlock()
	iph.backends = append(iph.backends, backend)
}

// RemoveBackend removes a backend from the pool.
func (iph *IPHashStrategy) RemoveBackend(backend *Backend) {
	iph.mutex.Lock()
	defer iph.mutex.Unlock()

	for i, b := range iph.backends {
		if b == backend {
			iph.backends[i] = iph.backends[len(iph.backends)-1]
			iph.backends = iph.backends[:len(iph.backends)-1]
			return
		}
	}
}

// GetBackends returns all backends in the pool.
func (iph *IPHashStrategy) GetBackends() []*Backend {
	iph.mutex.RLock()
	defer iph.mutex.RUnlock()

	backends := make([]*Backend, len(iph.backends))
	copy(backends, iph.backends)
	return backends
}
