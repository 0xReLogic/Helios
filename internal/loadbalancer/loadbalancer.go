package loadbalancer

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0xReLogic/Helios/internal/circuitbreaker"
	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/logging"
	"github.com/0xReLogic/Helios/internal/metrics"
	"github.com/0xReLogic/Helios/internal/ratelimiter"
	"github.com/0xReLogic/Helios/internal/utils"
)

// Strategy defines the interface for load balancing strategies
type Strategy interface {
	NextBackend(r *http.Request) *Backend
	AddBackend(backend *Backend)
	RemoveBackend(backend *Backend)
	GetBackends() []*Backend
}

// BackendInfo is a lightweight snapshot used by the Admin API
type BackendInfo struct {
	Name              string `json:"name"`
	Address           string `json:"address"`
	Healthy           bool   `json:"healthy"`
	ActiveConnections int32  `json:"active_connections"`
	Weight            int    `json:"weight"`
}

// ListBackends returns a snapshot of backends for the Admin API
func (lb *LoadBalancer) ListBackends() []BackendInfo {
	lb.mutex.RLock()
	backends := lb.strategy.GetBackends()
	lb.mutex.RUnlock()

	infos := make([]BackendInfo, 0, len(backends))
	for _, b := range backends {
		b.Mutex.RLock()
		info := BackendInfo{
			Name:              b.Name,
			Address:           b.URL.String(),
			Healthy:           b.IsHealthy,
			ActiveConnections: b.ActiveConnections,
			Weight:            b.Weight,
		}
		b.Mutex.RUnlock()
		infos = append(infos, info)
	}
	return infos
}

// SetStrategy switches the load balancing strategy at runtime
func (lb *LoadBalancer) SetStrategy(name string) error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	var newStrategy Strategy
	switch name {
	case "round_robin":
		newStrategy = NewRoundRobinStrategy()
	case "least_connections":
		newStrategy = NewLeastConnectionsStrategy()
	case "weighted_round_robin":
		newStrategy = NewWeightedRoundRobinStrategy()
	case "ip_hash":
		newStrategy = NewIPHashStrategy()
	default:
		return fmt.Errorf("unknown strategy: %s", name)
	}

	// Move existing backends to the new strategy
	for _, b := range lb.strategy.GetBackends() {
		newStrategy.AddBackend(b)
	}

	lb.strategy = newStrategy
	lb.config.LoadBalancer.Strategy = name
	logging.L().Info().Str("strategy", name).Msg("load balancing strategy switched")
	return nil
}

// Backend represents a backend server
type Backend struct {
	Name              string
	URL               *url.URL
	ReverseProxy      *httputil.ReverseProxy
	IsHealthy         bool
	UnhealthyUntil    time.Time    // Time until which the backend is considered unhealthy
	ActiveConnections int32        // Number of active connections
	Weight            int          // Weight for weighted load balancing strategies
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
	strategy         Strategy
	mutex            sync.RWMutex
	config           *config.Config
	healthChecks     *healthChecker
	rateLimiter      ratelimiter.RateLimiter
	circuitBreaker   *circuitbreaker.CircuitBreaker
	metricsCollector *metrics.MetricsCollector
	ctx              context.Context
	cancel           context.CancelFunc
	healthCheckWg    sync.WaitGroup
	wsPool           *WebSocketPool
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
	case "weighted_round_robin":
		strategy = NewWeightedRoundRobinStrategy()
	case "ip_hash":
		strategy = NewIPHashStrategy()
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

	// Create context for managing goroutines lifecycle
	ctx, cancel := context.WithCancel(context.Background())

	// Create the load balancer
	lb := &LoadBalancer{
		strategy:         strategy,
		config:           cfg,
		healthChecks:     healthChecks,
		metricsCollector: metrics.NewMetricsCollector(),
		ctx:              ctx,
		cancel:           cancel,
	}

	// Setup WebSocket connection pool if configured
	if cfg.LoadBalancer.WebSocketPool.Enabled {
		maxIdle := cfg.LoadBalancer.WebSocketPool.MaxIdle
		if maxIdle <= 0 {
			maxIdle = 10
		}
		maxActive := cfg.LoadBalancer.WebSocketPool.MaxActive
		if maxActive <= 0 {
			maxActive = 100
		}
		idleTimeout := time.Duration(cfg.LoadBalancer.WebSocketPool.IdleTimeoutSeconds) * time.Second
		if idleTimeout <= 0 {
			idleTimeout = 5 * time.Minute
		}

		lb.wsPool = NewWebSocketPool(maxIdle, maxActive, idleTimeout)
		logging.L().Info().
			Int("max_idle", maxIdle).
			Int("max_active", maxActive).
			Dur("idle_timeout", idleTimeout).
			Msg("WebSocket connection pool enabled")
	}

	// Setup rate limiting if enabled
	if cfg.RateLimit.Enabled {
		maxTokens := cfg.RateLimit.MaxTokens
		if maxTokens <= 0 {
			maxTokens = 100 // Default
		}
		refillRate := time.Duration(cfg.RateLimit.RefillRate) * time.Second
		if refillRate <= 0 {
			refillRate = time.Second // Default
		}
		lb.rateLimiter = ratelimiter.NewTokenBucketRateLimiter(maxTokens, refillRate)
		logging.L().Info().Int("max_tokens", maxTokens).Dur("refill_rate", refillRate).Msg("rate limiting enabled")
	}

	// Setup circuit breaker if enabled
	if cfg.CircuitBreaker.Enabled {
		cbSettings := circuitbreaker.Settings{
			Name:             "helios-lb",
			MaxRequests:      uint32(cfg.CircuitBreaker.MaxRequests),
			Interval:         time.Duration(cfg.CircuitBreaker.IntervalSeconds) * time.Second,
			Timeout:          time.Duration(cfg.CircuitBreaker.TimeoutSeconds) * time.Second,
			FailureThreshold: uint32(cfg.CircuitBreaker.FailureThreshold),
			SuccessThreshold: uint32(cfg.CircuitBreaker.SuccessThreshold),
			OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
				logging.L().Info().Str("circuit_breaker", name).Str("from", from.String()).Str("to", to.String()).Msg("circuit breaker state changed")
				// Update metrics
				failureCount, successCount, requestCount := lb.circuitBreaker.Counts()
				lb.metricsCollector.UpdateCircuitBreakerState(name, to.String(), failureCount, successCount, requestCount)
			},
		}

		// Set defaults if not provided
		if cbSettings.MaxRequests == 0 {
			cbSettings.MaxRequests = 1
		}
		if cbSettings.Interval == 0 {
			cbSettings.Interval = time.Minute
		}
		if cbSettings.Timeout == 0 {
			cbSettings.Timeout = time.Minute
		}
		if cbSettings.FailureThreshold == 0 {
			cbSettings.FailureThreshold = 5
		}
		if cbSettings.SuccessThreshold == 0 {
			cbSettings.SuccessThreshold = 1
		}

		lb.circuitBreaker = circuitbreaker.NewCircuitBreaker(cbSettings)
		logging.L().Info().Uint32("failure_threshold", cbSettings.FailureThreshold).Msg("circuit breaker enabled")
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
		logging.L().Info().Dur("interval", lb.healthChecks.activeInterval).Msg("active health checks enabled")
	} else {
		logging.L().Info().Msg("active health checks disabled")
	}

	if lb.healthChecks.passiveEnabled {
		logging.L().Info().Int("threshold", lb.healthChecks.passiveThreshold).Dur("timeout", lb.healthChecks.passiveTimeout).Msg("passive health checks enabled")
	} else {
		logging.L().Info().Msg("passive health checks disabled")
	}

	return lb, nil
}

// startActiveHealthChecks starts a goroutine that periodically checks the health of all backends
func (lb *LoadBalancer) startActiveHealthChecks() {
	ticker := time.NewTicker(lb.healthChecks.activeInterval)
	defer ticker.Stop()

	logging.L().Info().Dur("interval", lb.healthChecks.activeInterval).Msg("starting active health checks")

	// Run an initial health check immediately
	lb.checkBackendsHealth()

	// Monitor ticker and context cancellation
	for {
		select {
		case <-lb.ctx.Done():
			logging.L().Info().Msg("stopping active health checks")
			lb.healthCheckWg.Wait()
			return
		case <-ticker.C:
			lb.checkBackendsHealth()
		}
	}
}

// checkBackendsHealth checks the health of all backends
func (lb *LoadBalancer) checkBackendsHealth() {
	lb.mutex.RLock()
	backends := lb.strategy.GetBackends()
	lb.mutex.RUnlock()

	for _, backend := range backends {
		lb.healthCheckWg.Add(1)
		go func(b *Backend) {
			defer lb.healthCheckWg.Done()
			lb.checkBackendHealth(b)
		}(backend)
	}
}

// checkBackendHealth checks the health of a single backend
func (lb *LoadBalancer) checkBackendHealth(backend *Backend) {
	// Check if context is cancelled before starting health check
	select {
	case <-lb.ctx.Done():
		return
	default:
	}

	// Skip health check if the backend is already marked as unhealthy
	if !lb.IsBackendHealthy(backend) {
		return
	}

	// Create a health check request with context
	healthURL := *backend.URL
	healthURL.Path = lb.healthChecks.activePath

	req, err := http.NewRequestWithContext(lb.ctx, "GET", healthURL.String(), nil)
	if err != nil {
		logging.L().Error().Str("backend", backend.Name).Err(err).Msg("error creating health check request")
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
		logging.L().Error().Str("backend", backend.Name).Err(err).Msg("health check failed")
		lb.MarkBackendUnhealthy(backend, lb.healthChecks.passiveTimeout)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logging.L().Warn().Str("backend", backend.Name).Int("status", resp.StatusCode).Msg("health check returned non-ok status")
		lb.MarkBackendUnhealthy(backend, lb.healthChecks.passiveTimeout)
		return
	}

	// If we get here, the backend is healthy
	backend.Mutex.Lock()
	wasUnhealthy := !backend.IsHealthy
	backend.IsHealthy = true
	backend.Mutex.Unlock()

	// Update metrics to reflect healthy status
	if lb.metricsCollector != nil {
		lb.metricsCollector.UpdateBackendHealth(backend.Name, true)
	}

	if wasUnhealthy {
		logging.L().Info().Str("backend", backend.Name).Msg("backend marked healthy via active check")
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

	// Create a reverse proxy for this backend with optimized transport
	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Configure custom transport with timeouts (LEETCODE-STYLE OPTIMIZATION!)
	dialTimeout := time.Duration(lb.config.Server.Timeouts.BackendDial) * time.Second
	if dialTimeout == 0 {
		dialTimeout = 10 * time.Second // Default: 10s dial timeout
	}

	readTimeout := time.Duration(lb.config.Server.Timeouts.BackendRead) * time.Second
	if readTimeout == 0 {
		readTimeout = 30 * time.Second // Default: 30s backend read timeout
	}

	idleConnTimeout := time.Duration(lb.config.Server.Timeouts.BackendIdle) * time.Second
	if idleConnTimeout == 0 {
		idleConnTimeout = 90 * time.Second // Default: 90s idle connection timeout
	}

	// Custom transport with connection pooling and timeout optimization
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		// Connection pooling (prevent connection exhaustion)
		MaxIdleConns:        100, // Total idle connections
		MaxIdleConnsPerHost: 10,  // Per-host idle connections
		MaxConnsPerHost:     100, // Limit concurrent connections per host
		IdleConnTimeout:     idleConnTimeout,

		// Timeouts
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: readTimeout,
		ExpectContinueTimeout: 1 * time.Second,

		// Performance optimizations
		ForceAttemptHTTP2:  true,  // Use HTTP/2 when available
		DisableCompression: false, // Let backend handle compression
	}

	proxy.Transport = transport

	// Create the backend
	// If weight is not specified or is invalid, default to 1
	weight := backendCfg.Weight
	if weight < 1 {
		weight = 1
	}
	backend := &Backend{
		Name:              backendCfg.Name,
		URL:               backendURL,
		ReverseProxy:      proxy,
		IsHealthy:         true,        // Assume healthy initially
		UnhealthyUntil:    time.Time{}, // Zero time means it's healthy
		ActiveConnections: 0,
		Weight:            weight,
	}

	// Add to the strategy
	lb.strategy.AddBackend(backend)

	// Initialize metrics for backend health
	if lb.metricsCollector != nil {
		lb.metricsCollector.UpdateBackendHealth(backend.Name, backend.IsHealthy)
	}

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
func (lb *LoadBalancer) NextBackend(r *http.Request) *Backend {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	return lb.strategy.NextBackend(r)
}

// MarkBackendUnhealthy marks a backend as unhealthy for a specified duration
func (lb *LoadBalancer) MarkBackendUnhealthy(backend *Backend, duration time.Duration) {
	backend.Mutex.Lock()
	defer backend.Mutex.Unlock()

	backend.IsHealthy = false
	backend.UnhealthyUntil = time.Now().Add(duration)

	// Update metrics to reflect unhealthy status
	if lb.metricsCollector != nil {
		lb.metricsCollector.UpdateBackendHealth(backend.Name, false)
	}

	logging.L().Warn().Str("backend", backend.Name).Dur("unhealthy_for", duration).Msg("backend marked unhealthy")
}

// IsBackendHealthy checks if a backend is currently healthy
func (lb *LoadBalancer) IsBackendHealthy(backend *Backend) bool {
	backend.Mutex.RLock()
	isHealthy := backend.IsHealthy
	unhealthyUntil := backend.UnhealthyUntil
	backend.Mutex.RUnlock()

	// If it's marked as unhealthy, check if the unhealthy period has expired
	if !isHealthy && time.Now().After(unhealthyUntil) {
		// The unhealthy period has expired, mark it as healthy again
		backend.Mutex.Lock()
		// Double-check after acquiring write lock to prevent race condition
		if !backend.IsHealthy && time.Now().After(backend.UnhealthyUntil) {
			backend.IsHealthy = true
			backend.Mutex.Unlock()

			// Update metrics to reflect healthy status
			if lb.metricsCollector != nil {
				lb.metricsCollector.UpdateBackendHealth(backend.Name, true)
			}

			logging.L().Info().Str("backend", backend.Name).Msg("backend marked healthy")
			return true
		}
		backend.Mutex.Unlock()
		return false
	}

	return isHealthy
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

// GetMetricsCollector returns the metrics collector
func (lb *LoadBalancer) GetMetricsCollector() *metrics.MetricsCollector {
	return lb.metricsCollector
}

// ServeHTTP implements the http.Handler interface
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	logger := logging.WithContext(r.Context())

	// Record the request
	lb.metricsCollector.RecordRequest()

	// Check rate limiting if enabled
	if lb.rateLimiter != nil {
		clientIP := utils.GetClientIP(r)
		if !lb.rateLimiter.Allow(clientIP) {
			lb.metricsCollector.RecordRateLimitedRequest()
			logger.Warn().Str("client_ip", clientIP).Msg("request rate limited")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
	}

	// Execute request with circuit breaker protection if enabled
	if lb.circuitBreaker != nil {
		err := lb.circuitBreaker.Execute(func() error {
			return lb.handleRequest(w, r, startTime)
		})
		if err != nil {
			failureCount, successCount, requestCount := lb.circuitBreaker.Counts()
			logger.Error().
				Err(err).
				Uint32("failure_count", failureCount).
				Uint32("success_count", successCount).
				Uint32("total_requests", requestCount).
				Msg("circuit breaker execution failed")

			if err == circuitbreaker.ErrCircuitBreakerOpen {
				http.Error(w, fmt.Sprintf("Service temporarily unavailable - circuit breaker is open (failures: %d, requests: %d)", failureCount, requestCount), http.StatusServiceUnavailable)
			} else if err == circuitbreaker.ErrTooManyRequests {
				http.Error(w, fmt.Sprintf("Too many requests - circuit breaker half-open (successes: %d)", successCount), http.StatusTooManyRequests)
			} else {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			lb.metricsCollector.RecordResponse(false, time.Since(startTime))
			return
		}
	} else {
		// Execute without circuit breaker
		if err := lb.handleRequest(w, r, startTime); err != nil {
			logger.Error().Err(err).Msg("request handling failed")
		}
	}
}

// handleRequest handles the actual request processing
func (lb *LoadBalancer) handleRequest(w http.ResponseWriter, r *http.Request, startTime time.Time) error {
	// Find a healthy backend
	var backend *Backend
	for i := 0; i < 3; i++ { // Try up to 3 times to find a healthy backend
		backend = lb.NextBackend(r)
		if backend == nil {
			http.Error(w, "No available backend servers", http.StatusServiceUnavailable)
			return nil
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
		logging.WithContext(r.Context()).Warn().Str("path", r.URL.Path).Msg("no healthy backend available")
		http.Error(w, "No healthy backend servers available", http.StatusServiceUnavailable)
		return nil
	}

	// Track the active connection
	backend.IncrementConnections()
	lb.metricsCollector.UpdateBackendConnections(backend.Name, backend.GetActiveConnections())

	// Create a custom response writer to capture the status code
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status code
	}

	// Forward the request to the selected backend
	backend.ReverseProxy.ServeHTTP(rw, r)

	// Decrement the connection count when done
	backend.DecrementConnections()
	lb.metricsCollector.UpdateBackendConnections(backend.Name, backend.GetActiveConnections())

	// Record metrics
	responseTime := time.Since(startTime)
	success := rw.statusCode < 500
	lb.metricsCollector.RecordResponse(success, responseTime)
	lb.metricsCollector.RecordBackendRequest(backend.Name, success, responseTime)

	logger := logging.WithContext(r.Context())

	// Check if the backend returned an error status code (5xx) and passive health checks are enabled
	if rw.statusCode >= 500 && lb.healthChecks.passiveEnabled {
		// Increment failure count for this backend
		lb.healthChecks.unhealthyBackendMu.Lock()
		lb.healthChecks.unhealthyBackends[backend.Name]++
		failureCount := lb.healthChecks.unhealthyBackends[backend.Name]
		lb.healthChecks.unhealthyBackendMu.Unlock()

		logger.Warn().Str("backend", backend.Name).
			Int("status", rw.statusCode).
			Int("failure_count", failureCount).
			Int("threshold", lb.healthChecks.passiveThreshold).
			Msg("backend returned server error")

		// If failure count exceeds threshold, mark as unhealthy
		if failureCount >= lb.healthChecks.passiveThreshold {
			lb.MarkBackendUnhealthy(backend, lb.healthChecks.passiveTimeout)

			// Reset failure count
			lb.healthChecks.unhealthyBackendMu.Lock()
			lb.healthChecks.unhealthyBackends[backend.Name] = 0
			lb.healthChecks.unhealthyBackendMu.Unlock()
		}

		return circuitbreaker.ErrCircuitBreakerOpen // Return error for circuit breaker
	}

	latencyMs := float64(responseTime) / float64(time.Millisecond)
	logger.Info().
		Str("backend", backend.Name).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Int("status", rw.statusCode).
		Float64("latency_ms", latencyMs).
		Msg("request completed")

	return nil
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

// Hijack implements the http.Hijacker interface to support websockets
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
	}
	return h.Hijack()
}

// Stop gracefully shuts down the load balancer and waits for all health check goroutines to finish
func (lb *LoadBalancer) Stop() {
	logging.L().Info().Msg("shutting down load balancer")
	lb.cancel()
	lb.healthCheckWg.Wait()

	// Shutdown WebSocket pool if enabled
	if lb.wsPool != nil {
		lb.wsPool.Shutdown()
		logging.L().Info().Msg("WebSocket connection pool shutdown complete")
	}

	logging.L().Info().Msg("load balancer shutdown complete")
}
