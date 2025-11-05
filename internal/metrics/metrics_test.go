package metrics

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()

	// Record some requests
	mc.RecordRequest()
	mc.RecordResponse(true, 100*time.Millisecond)

	mc.RecordRequest()
	mc.RecordResponse(false, 200*time.Millisecond)

	// Get metrics
	metrics := mc.GetMetrics()

	// Verify total requests
	if metrics.TotalRequests != 2 {
		t.Errorf("Expected 2 total requests, got %d", metrics.TotalRequests)
	}

	// Verify successful requests
	if metrics.SuccessfulRequests != 1 {
		t.Errorf("Expected 1 successful request, got %d", metrics.SuccessfulRequests)
	}

	// Verify failed requests
	if metrics.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", metrics.FailedRequests)
	}

	// Verify average response time (using EMA)
	// With alpha=0.2: EMA = 0.2*new + 0.8*old
	// First: EMA = 100
	// Second: EMA = 0.2*200 + 0.8*100 = 40 + 80 = 120
	expectedAvg := 120.0
	if metrics.AverageResponseTime != expectedAvg {
		t.Errorf("Expected average response time %.1f, got %.1f", expectedAvg, metrics.AverageResponseTime)
	}
}

func TestExponentialMovingAverage(t *testing.T) {
	mc := NewMetricsCollector()

	// Test EMA behavior with multiple requests
	mc.RecordRequest()
	mc.RecordResponse(true, 100*time.Millisecond)
	
	mc.RecordRequest()
	mc.RecordResponse(true, 200*time.Millisecond)
	
	mc.RecordRequest()
	mc.RecordResponse(true, 300*time.Millisecond)

	metrics := mc.GetMetrics()

	// With alpha=0.2:
	// 1st: EMA = 100
	// 2nd: EMA = 0.2*200 + 0.8*100 = 40 + 80 = 120
	// 3rd: EMA = 0.2*300 + 0.8*120 = 60 + 96 = 156
	expectedAvg := 156.0
	if metrics.AverageResponseTime != expectedAvg {
		t.Errorf("Expected EMA %.1f, got %.1f", expectedAvg, metrics.AverageResponseTime)
	}
}

func TestBackendEMA(t *testing.T) {
	mc := NewMetricsCollector()

	// Test backend-specific EMA
	mc.RecordBackendRequest("test", true, 100*time.Millisecond)
	mc.RecordBackendRequest("test", true, 300*time.Millisecond)

	metrics := mc.GetMetrics()
	backend := metrics.BackendMetrics["test"]

	// With alpha=0.2:
	// 1st: EMA = 100
	// 2nd: EMA = 0.2*300 + 0.8*100 = 60 + 80 = 140
	expectedAvg := 140.0
	if backend.AverageResponseTime != expectedAvg {
		t.Errorf("Expected backend EMA %.1f, got %.1f", expectedAvg, backend.AverageResponseTime)
	}
}

func TestNoMemoryOverflow(t *testing.T) {
	mc := NewMetricsCollector()

	// Simulate many requests to verify no overflow
	for i := 0; i < 10000; i++ {
		mc.RecordRequest()
		mc.RecordResponse(true, 50*time.Millisecond)
	}

	metrics := mc.GetMetrics()

	// Average should converge to ~50ms due to EMA
	if metrics.AverageResponseTime < 45 || metrics.AverageResponseTime > 55 {
		t.Errorf("Expected EMA to converge to ~50ms, got %.1fms", metrics.AverageResponseTime)
	}

	// Ensure total requests is counted correctly
	if metrics.TotalRequests != 10000 {
		t.Errorf("Expected 10000 requests, got %d", metrics.TotalRequests)
	}
}

func TestMaxBackendsLimit(t *testing.T) {
	mc := NewMetricsCollector()

	// Try to add more than MaxBackendMetrics
	for i := 0; i < MaxBackendMetrics+100; i++ {
		backendName := "backend-" + string(rune(i))
		mc.RecordBackendRequest(backendName, true, 50*time.Millisecond)
	}

	metrics := mc.GetMetrics()

	// Should not exceed max limit
	if len(metrics.BackendMetrics) > MaxBackendMetrics {
		t.Errorf("Backend metrics exceeded limit: got %d, max %d", 
			len(metrics.BackendMetrics), MaxBackendMetrics)
	}
}

func TestMaxCircuitBreakerLimit(t *testing.T) {
	mc := NewMetricsCollector()

	// Try to add more than MaxCircuitBreakerMetrics
	for i := 0; i < MaxCircuitBreakerMetrics+10; i++ {
		cbName := "cb-" + string(rune(i))
		mc.UpdateCircuitBreakerState(cbName, "closed", 0, 0, 0)
	}

	metrics := mc.GetMetrics()

	// Should not exceed max limit
	if len(metrics.CircuitBreakerMetrics) > MaxCircuitBreakerMetrics {
		t.Errorf("Circuit breaker metrics exceeded limit: got %d, max %d",
			len(metrics.CircuitBreakerMetrics), MaxCircuitBreakerMetrics)
	}
}

func TestConcurrentMetricsAccess(t *testing.T) {
	mc := NewMetricsCollector()
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				mc.RecordRequest()
				mc.RecordResponse(true, time.Duration(j)*time.Millisecond)
				mc.RecordBackendRequest("backend-1", true, 50*time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			_ = mc.GetMetrics()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 11; i++ {
		<-done
	}

	metrics := mc.GetMetrics()

	// Verify metrics were collected correctly
	if metrics.TotalRequests != 1000 {
		t.Errorf("Expected 1000 total requests, got %d", metrics.TotalRequests)
	}
}

func TestBackendMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	// Record backend requests
	mc.RecordBackendRequest("backend1", true, 50*time.Millisecond)
	mc.RecordBackendRequest("backend1", false, 150*time.Millisecond)
	mc.RecordBackendRequest("backend2", true, 100*time.Millisecond)

	metrics := mc.GetMetrics()

	// Check backend1 metrics
	backend1, exists := metrics.BackendMetrics["backend1"]
	if !exists {
		t.Fatal("backend1 metrics should exist")
	}

	if backend1.TotalRequests != 2 {
		t.Errorf("Expected 2 requests for backend1, got %d", backend1.TotalRequests)
	}

	if backend1.SuccessfulRequests != 1 {
		t.Errorf("Expected 1 successful request for backend1, got %d", backend1.SuccessfulRequests)
	}

	if backend1.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request for backend1, got %d", backend1.FailedRequests)
	}

	// Check backend2 metrics
	backend2, exists := metrics.BackendMetrics["backend2"]
	if !exists {
		t.Fatal("backend2 metrics should exist")
	}

	if backend2.TotalRequests != 1 {
		t.Errorf("Expected 1 request for backend2, got %d", backend2.TotalRequests)
	}
}

func TestRateLimitMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	// Record rate limited requests
	mc.RecordRateLimitedRequest()
	mc.RecordRateLimitedRequest()

	metrics := mc.GetMetrics()

	if metrics.RateLimitedRequests != 2 {
		t.Errorf("Expected 2 rate limited requests, got %d", metrics.RateLimitedRequests)
	}
}

func TestMetricsHandler(t *testing.T) {
	mc := NewMetricsCollector()

	// Add some test data
	mc.RecordRequest()
	mc.RecordResponse(true, 100*time.Millisecond)

	// Create test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler := mc.MetricsHandler()
	handler(w, req)

	// Check response
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse JSON response
	var metrics Metrics
	err := json.Unmarshal(w.Body.Bytes(), &metrics)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Verify data
	if metrics.TotalRequests != 1 {
		t.Errorf("Expected 1 total request in response, got %d", metrics.TotalRequests)
	}
}

func TestHealthHandler(t *testing.T) {
	mc := NewMetricsCollector()

	// Create test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler := mc.HealthHandler()
	handler(w, req)

	// Check response
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse JSON response
	var health map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &health)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Verify status
	if health["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", health["status"])
	}
}
