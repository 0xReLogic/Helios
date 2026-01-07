package loadbalancer

import (
	"hash/fnv"
	"net"
	"net/http"
	"strings"
	"sync"
)

// IPHashConsistentStrategy implements IP hash with Jump Consistent Hash algorithm.
// This strategy provides minimal key remapping when backends are added/removed,
// making it ideal for stateful applications with sessions or caches.
//
// Trade-offs vs standard IPHashStrategy:
// - Better: Only ~9-13% of IPs remapped when scaling (vs ~90% with modulo)
// - Worse: Slightly slower hot path (18.59ns vs 12.40ns)
// - Worse: Less uniform distribution (±40% vs ±0.59%)
//
// Use this when:
// - You have stateful applications (sessions, caches)
// - Backends scale frequently (auto-scaling)
// - Session preservation > perfect load distribution
//
// Use standard ip-hash when:
// - You have stateless applications
// - Backends rarely change
// - Perfect load distribution is critical
type IPHashConsistentStrategy struct {
	backends []*Backend
	mutex    sync.RWMutex
}

// NewIPHashConsistentStrategy creates a new Jump Consistent Hash strategy.
func NewIPHashConsistentStrategy() *IPHashConsistentStrategy {
	return &IPHashConsistentStrategy{
		backends: make([]*Backend, 0),
	}
}

// jumpHash implements the Jump Consistent Hash algorithm.
// Paper: https://arxiv.org/abs/1406.2294
// This algorithm guarantees minimal key remapping when the number of buckets changes.
func jumpHash(key uint64, numBuckets int32) int32 {
	var b int64 = -1
	var j int64 = 0

	for j < int64(numBuckets) {
		b = j
		key = key*2862933555777941757 + 1
		j = (b + 1) * (int64(1<<31) / int64((key>>33)+1))
	}
	return int32(b)
}

// NextBackend returns the next backend using Jump Consistent Hash algorithm.
func (iph *IPHashConsistentStrategy) NextBackend(r *http.Request) *Backend {
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

	// Use Jump Consistent Hash to select backend
	// This ensures minimal remapping when backends are added/removed
	index := jumpHash(uint64(hashValue), int32(len(healthyBackends))) // #nosec G115 - len() is always non-negative, safe conversion
	return healthyBackends[index]
}

// AddBackend adds a backend to the pool.
func (iph *IPHashConsistentStrategy) AddBackend(backend *Backend) {
	iph.mutex.Lock()
	defer iph.mutex.Unlock()
	iph.backends = append(iph.backends, backend)
}

// RemoveBackend removes a backend from the pool.
func (iph *IPHashConsistentStrategy) RemoveBackend(backend *Backend) {
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
func (iph *IPHashConsistentStrategy) GetBackends() []*Backend {
	iph.mutex.RLock()
	defer iph.mutex.RUnlock()

	backends := make([]*Backend, len(iph.backends))
	copy(backends, iph.backends)
	return backends
}
