package metrics

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds all the metrics for the load balancer
type Metrics struct {
	// Request metrics
	TotalRequests      uint64 `json:"total_requests"`
	SuccessfulRequests uint64 `json:"successful_requests"`
	FailedRequests     uint64 `json:"failed_requests"`

	// Response time metrics
	TotalResponseTime   uint64  `json:"total_response_time_ms"`
	AverageResponseTime float64 `json:"average_response_time_ms"`

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
	TotalResponseTime   uint64    `json:"total_response_time_ms"`
	AverageResponseTime float64   `json:"average_response_time_ms"`
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
	metrics *Metrics
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: &Metrics{
			BackendMetrics:        make(map[string]*BackendMetrics),
			CircuitBreakerMetrics: make(map[string]*CircuitBreakerMetrics),
			StartTime:             time.Now(),
		},
	}
}

// RecordRequest records a new request
func (mc *MetricsCollector) RecordRequest() {
	atomic.AddUint64(&mc.metrics.TotalRequests, 1)
}

// RecordResponse records a response with its status and duration
func (mc *MetricsCollector) RecordResponse(success bool, responseTime time.Duration) {
	responseTimeMs := uint64(responseTime.Milliseconds())

	if success {
		atomic.AddUint64(&mc.metrics.SuccessfulRequests, 1)
	} else {
		atomic.AddUint64(&mc.metrics.FailedRequests, 1)
	}

	atomic.AddUint64(&mc.metrics.TotalResponseTime, responseTimeMs)

	// Update average response time
	mc.updateAverageResponseTime()
}

// RecordBackendRequest records a request to a specific backend
func (mc *MetricsCollector) RecordBackendRequest(backendName string, success bool, responseTime time.Duration) {
	mc.metrics.mutex.Lock()
	defer mc.metrics.mutex.Unlock()

	backend, exists := mc.metrics.BackendMetrics[backendName]
	if !exists {
		backend = &BackendMetrics{
			Name: backendName,
		}
		mc.metrics.BackendMetrics[backendName] = backend
	}

	backend.TotalRequests++
	responseTimeMs := uint64(responseTime.Milliseconds())
	backend.TotalResponseTime += responseTimeMs

	if success {
		backend.SuccessfulRequests++
	} else {
		backend.FailedRequests++
	}

	// Update average response time for backend
	if backend.TotalRequests > 0 {
		backend.AverageResponseTime = float64(backend.TotalResponseTime) / float64(backend.TotalRequests)
	}
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
	defer mc.metrics.mutex.Unlock()

	cb, exists := mc.metrics.CircuitBreakerMetrics[name]
	if !exists {
		cb = &CircuitBreakerMetrics{
			Name: name,
		}
		mc.metrics.CircuitBreakerMetrics[name] = cb
	}

	// Update state change time if state changed
	if cb.State != state {
		cb.LastStateChange = time.Now()
	}

	cb.State = state
	cb.FailureCount = failureCount
	cb.SuccessCount = successCount
	cb.RequestCount = requestCount
}

// updateAverageResponseTime calculates the average response time
func (mc *MetricsCollector) updateAverageResponseTime() {
	totalRequests := atomic.LoadUint64(&mc.metrics.TotalRequests)
	if totalRequests > 0 {
		totalResponseTime := atomic.LoadUint64(&mc.metrics.TotalResponseTime)
		mc.metrics.AverageResponseTime = float64(totalResponseTime) / float64(totalRequests)
	}
}

// GetMetrics returns a copy of current metrics
func (mc *MetricsCollector) GetMetrics() *Metrics {
	mc.metrics.mutex.RLock()
	defer mc.metrics.mutex.RUnlock()

	// Update uptime
	mc.metrics.Uptime = time.Since(mc.metrics.StartTime).String()

	// Create a deep copy
	metricsCopy := &Metrics{
		TotalRequests:         atomic.LoadUint64(&mc.metrics.TotalRequests),
		SuccessfulRequests:    atomic.LoadUint64(&mc.metrics.SuccessfulRequests),
		FailedRequests:        atomic.LoadUint64(&mc.metrics.FailedRequests),
		TotalResponseTime:     atomic.LoadUint64(&mc.metrics.TotalResponseTime),
		AverageResponseTime:   mc.metrics.AverageResponseTime,
		RateLimitedRequests:   atomic.LoadUint64(&mc.metrics.RateLimitedRequests),
		StartTime:             mc.metrics.StartTime,
		Uptime:                mc.metrics.Uptime,
		BackendMetrics:        make(map[string]*BackendMetrics),
		CircuitBreakerMetrics: make(map[string]*CircuitBreakerMetrics),
	}

	// Copy backend metrics
	for name, backend := range mc.metrics.BackendMetrics {
		metricsCopy.BackendMetrics[name] = &BackendMetrics{
			Name:                backend.Name,
			TotalRequests:       backend.TotalRequests,
			SuccessfulRequests:  backend.SuccessfulRequests,
			FailedRequests:      backend.FailedRequests,
			ActiveConnections:   backend.ActiveConnections,
			TotalResponseTime:   backend.TotalResponseTime,
			AverageResponseTime: backend.AverageResponseTime,
			IsHealthy:           backend.IsHealthy,
			LastHealthCheck:     backend.LastHealthCheck,
		}
	}

	// Copy circuit breaker metrics
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
