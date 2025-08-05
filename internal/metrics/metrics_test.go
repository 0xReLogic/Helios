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

	// Verify average response time
	expectedAvg := (100.0 + 200.0) / 2.0 // 150ms
	if metrics.AverageResponseTime != expectedAvg {
		t.Errorf("Expected average response time %.1f, got %.1f", expectedAvg, metrics.AverageResponseTime)
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
