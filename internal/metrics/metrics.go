package metrics

import (
	"encoding/json"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Maximum number of backend metrics to prevent unbounded memory growth
	MaxBackendMetrics = 1000
	// Maximum number of circuit breaker metrics
	MaxCircuitBreakerMetrics = 100
	// EMA smoothing factor (20% weight to new samples)
	DefaultAlpha = 0.2
)

// Metrics holds all the metrics for the load balancer
type Metrics struct {
	// Request metrics (atomic counters)
	TotalRequests      uint64 `json:"total_requests"`
	SuccessfulRequests uint64 `json:"successful_requests"`
	FailedRequests     uint64 `json:"failed_requests"`

	// Response time metrics (using exponential moving average to prevent overflow)
	// Stored as uint64 bits of float64 for atomic operations
	avgResponseTimeBits uint64  // atomic access via math.Float64bits/math.Float64frombits
	AverageResponseTime float64 `json:"average_response_time_ms"` // for JSON serialization
	alpha               float64 // EMA smoothing factor (not exported)

	// Backend metrics
	BackendMetrics map[string]*BackendMetrics `json:"backend_metrics"`

	// Rate limiting metrics
	RateLimitedRequests uint64 `json:"rate_limited_requests"`

	// Circuit breaker metrics
	CircuitBreakerMetrics map[string]*CircuitBreakerMetrics `json:"circuit_breaker_metrics"`

	// System metrics
	StartTime time.Time `json:"start_time"`
	Uptime    string    `json:"uptime"`

	mutex sync.RWMutex
}

// BackendMetrics holds metrics for individual backends
type BackendMetrics struct {
	Name                string    `json:"name"`
	TotalRequests       uint64    `json:"total_requests"`
	SuccessfulRequests  uint64    `json:"successful_requests"`
	FailedRequests      uint64    `json:"failed_requests"`
	ActiveConnections   int32     `json:"active_connections"`
	AverageResponseTime float64   `json:"average_response_time_ms"`
	alpha               float64   // EMA smoothing factor (not exported)
	IsHealthy           bool      `json:"is_healthy"`
	LastHealthCheck     time.Time `json:"last_health_check"`
}

// CircuitBreakerMetrics holds metrics for circuit breakers
type CircuitBreakerMetrics struct {
	Name            string    `json:"name"`
	State           string    `json:"state"`
	FailureCount    uint32    `json:"failure_count"`
	SuccessCount    uint32    `json:"success_count"`
	RequestCount    uint32    `json:"request_count"`
	LastStateChange time.Time `json:"last_state_change"`
}

// MetricsCollector manages metrics collection
type MetricsCollector struct {
	metrics     *Metrics
	metricsPool sync.Pool // Pool for Metrics copies to reduce GC pressure
	backendPool sync.Pool // Pool for BackendMetrics copies
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		metrics: &Metrics{
			BackendMetrics:        make(map[string]*BackendMetrics),
			CircuitBreakerMetrics: make(map[string]*CircuitBreakerMetrics),
			StartTime:             time.Now(),
			alpha:                 DefaultAlpha,
		},
	}

	// Initialize object pools for zero-allocation copies
	mc.metricsPool.New = func() interface{} {
		return &Metrics{
			BackendMetrics:        make(map[string]*BackendMetrics),
			CircuitBreakerMetrics: make(map[string]*CircuitBreakerMetrics),
		}
	}

	mc.backendPool.New = func() interface{} {
		return &BackendMetrics{}
	}

	return mc
}

// RecordRequest records a new request
func (mc *MetricsCollector) RecordRequest() {
	atomic.AddUint64(&mc.metrics.TotalRequests, 1)
}

// RecordResponse records a response with its status and duration
func (mc *MetricsCollector) RecordResponse(success bool, responseTime time.Duration) {
	responseTimeMs := responseTime.Milliseconds()

	if success {
		atomic.AddUint64(&mc.metrics.SuccessfulRequests, 1)
	} else {
		atomic.AddUint64(&mc.metrics.FailedRequests, 1)
	}

	// Update average response time using Exponential Moving Average (EMA)
	// This prevents overflow and provides recent-weighted average
	mc.updateAverageResponseTime(float64(responseTimeMs))
}

// RecordBackendRequest records a request to a specific backend
func (mc *MetricsCollector) RecordBackendRequest(backendName string, success bool, responseTime time.Duration) {
	mc.metrics.mutex.Lock()

	// Check if we're exceeding max backends limit
	if len(mc.metrics.BackendMetrics) >= MaxBackendMetrics {
		mc.metrics.mutex.Unlock()
		return // Drop metric to prevent unbounded growth
	}

	backend, exists := mc.metrics.BackendMetrics[backendName]
	if !exists {
		backend = &BackendMetrics{
			Name:  backendName,
			alpha: DefaultAlpha,
		}
		mc.metrics.BackendMetrics[backendName] = backend
	}

	backend.TotalRequests++
	responseTimeMs := float64(responseTime.Milliseconds())

	if success {
		backend.SuccessfulRequests++
	} else {
		backend.FailedRequests++
	}

	// Update average response time using EMA (branchless for better perf)
	isFirst := backend.AverageResponseTime == 0
	backend.AverageResponseTime = float64(boolToInt(isFirst))*responseTimeMs +
		float64(1-boolToInt(isFirst))*(backend.alpha*responseTimeMs+(1-backend.alpha)*backend.AverageResponseTime)

	mc.metrics.mutex.Unlock()
}

// branchless conversion helper
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// UpdateBackendHealth updates the health status of a backend
func (mc *MetricsCollector) UpdateBackendHealth(backendName string, isHealthy bool) {
	mc.metrics.mutex.Lock()
	defer mc.metrics.mutex.Unlock()

	backend, exists := mc.metrics.BackendMetrics[backendName]
	if !exists {
		backend = &BackendMetrics{
			Name: backendName,
		}
		mc.metrics.BackendMetrics[backendName] = backend
	}

	backend.IsHealthy = isHealthy
	backend.LastHealthCheck = time.Now()
}

// UpdateBackendConnections updates the active connections count for a backend
func (mc *MetricsCollector) UpdateBackendConnections(backendName string, connections int32) {
	mc.metrics.mutex.Lock()
	defer mc.metrics.mutex.Unlock()

	backend, exists := mc.metrics.BackendMetrics[backendName]
	if !exists {
		backend = &BackendMetrics{
			Name: backendName,
		}
		mc.metrics.BackendMetrics[backendName] = backend
	}

	backend.ActiveConnections = connections
}

// RecordRateLimitedRequest records a rate-limited request
func (mc *MetricsCollector) RecordRateLimitedRequest() {
	atomic.AddUint64(&mc.metrics.RateLimitedRequests, 1)
}

// UpdateCircuitBreakerState updates the state of a circuit breaker
func (mc *MetricsCollector) UpdateCircuitBreakerState(name, state string, failureCount, successCount, requestCount uint32) {
	mc.metrics.mutex.Lock()

	// Prevent unbounded growth of circuit breaker metrics
	if len(mc.metrics.CircuitBreakerMetrics) >= MaxCircuitBreakerMetrics {
		mc.metrics.mutex.Unlock()
		return
	}

	cbMetrics, exists := mc.metrics.CircuitBreakerMetrics[name]
	if !exists {
		cbMetrics = &CircuitBreakerMetrics{
			Name: name,
		}
		mc.metrics.CircuitBreakerMetrics[name] = cbMetrics
	}

	cbMetrics.State = state
	cbMetrics.FailureCount = failureCount
	cbMetrics.SuccessCount = successCount
	cbMetrics.RequestCount = requestCount
	cbMetrics.LastStateChange = time.Now()

	mc.metrics.mutex.Unlock()
}

// updateAverageResponseTime calculates the average response time
func (mc *MetricsCollector) updateAverageResponseTime(newResponseTime float64) {
	// Lock-free atomic update using CAS loop
	for {
		oldBits := atomic.LoadUint64(&mc.metrics.avgResponseTimeBits)
		oldAvg := math.Float64frombits(oldBits)

		// Calculate new EMA
		var newAvg float64
		if oldAvg == 0 {
			newAvg = newResponseTime
		} else {
			newAvg = mc.metrics.alpha*newResponseTime + (1-mc.metrics.alpha)*oldAvg
		}

		newBits := math.Float64bits(newAvg)

		// Try to swap - retry if another goroutine updated it
		if atomic.CompareAndSwapUint64(&mc.metrics.avgResponseTimeBits, oldBits, newBits) {
			break
		}
	}
}

// GetMetrics returns a copy of current metrics
func (mc *MetricsCollector) GetMetrics() *Metrics {
	mc.metrics.mutex.RLock()

	// Update uptime (fast string operation)
	mc.metrics.Uptime = time.Since(mc.metrics.StartTime).String()

	// Get pooled metrics object to reduce allocations
	metricsCopy := mc.metricsPool.Get().(*Metrics)

	// Reset maps (reuse existing capacity)
	for k := range metricsCopy.BackendMetrics {
		delete(metricsCopy.BackendMetrics, k)
	}
	for k := range metricsCopy.CircuitBreakerMetrics {
		delete(metricsCopy.CircuitBreakerMetrics, k)
	}

	// Copy atomic counters (lock-free reads)
	metricsCopy.TotalRequests = atomic.LoadUint64(&mc.metrics.TotalRequests)
	metricsCopy.SuccessfulRequests = atomic.LoadUint64(&mc.metrics.SuccessfulRequests)
	metricsCopy.FailedRequests = atomic.LoadUint64(&mc.metrics.FailedRequests)
	metricsCopy.RateLimitedRequests = atomic.LoadUint64(&mc.metrics.RateLimitedRequests)

	// Copy average response time atomically
	avgBits := atomic.LoadUint64(&mc.metrics.avgResponseTimeBits)
	metricsCopy.AverageResponseTime = math.Float64frombits(avgBits)

	// Copy non-atomic fields
	metricsCopy.StartTime = mc.metrics.StartTime
	metricsCopy.Uptime = mc.metrics.Uptime

	// Copy backend metrics using pooled objects
	for name, backend := range mc.metrics.BackendMetrics {
		backendCopy := mc.backendPool.Get().(*BackendMetrics)
		backendCopy.Name = backend.Name
		backendCopy.TotalRequests = backend.TotalRequests
		backendCopy.SuccessfulRequests = backend.SuccessfulRequests
		backendCopy.FailedRequests = backend.FailedRequests
		backendCopy.ActiveConnections = backend.ActiveConnections
		backendCopy.AverageResponseTime = backend.AverageResponseTime
		backendCopy.IsHealthy = backend.IsHealthy
		backendCopy.LastHealthCheck = backend.LastHealthCheck
		metricsCopy.BackendMetrics[name] = backendCopy
	}

	// Copy circuit breaker metrics (usually small, direct allocation OK)
	for name, cb := range mc.metrics.CircuitBreakerMetrics {
		metricsCopy.CircuitBreakerMetrics[name] = &CircuitBreakerMetrics{
			Name:            cb.Name,
			State:           cb.State,
			FailureCount:    cb.FailureCount,
			SuccessCount:    cb.SuccessCount,
			RequestCount:    cb.RequestCount,
			LastStateChange: cb.LastStateChange,
		}
	}

	mc.metrics.mutex.RUnlock()

	return metricsCopy
}

// MetricsHandler returns an HTTP handler for the metrics endpoint
func (mc *MetricsCollector) MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := mc.GetMetrics()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(metrics); err != nil {
			http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
			return
		}
	}
}

// HealthHandler returns an HTTP handler for the health endpoint
func (mc *MetricsCollector) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := mc.GetMetrics()

		health := map[string]interface{}{
			"status":         "healthy",
			"uptime":         metrics.Uptime,
			"total_requests": metrics.TotalRequests,
			"backends":       make(map[string]interface{}),
		}

		// Add backend health status
		for name, backend := range metrics.BackendMetrics {
			health["backends"].(map[string]interface{})[name] = map[string]interface{}{
				"healthy":            backend.IsHealthy,
				"active_connections": backend.ActiveConnections,
				"last_health_check":  backend.LastHealthCheck,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(health); err != nil {
			http.Error(w, "Failed to encode health status", http.StatusInternalServerError)
			return
		}
	}
}
