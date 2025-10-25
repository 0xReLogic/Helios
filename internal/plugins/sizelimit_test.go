package plugins

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSizeLimitPlugin_RequestBodyExceedsLimit(t *testing.T) {
	// Create a mock next handler that reads the request body
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read the body
		_, err := io.ReadAll(r.Body)
		if err != nil {
			// MaxBytesReader should trigger an error
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Create the plugin with a 100-byte request limit
	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body":  100,
		"max_response_body": 1000,
	})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	// Create a request with body larger than 100 bytes
	largeBody := strings.Repeat("a", 200) // 200 bytes
	req := httptest.NewRequest("POST", "/test", strings.NewReader(largeBody))
	rec := httptest.NewRecorder()

	// Execute the middleware
	mw(handler).ServeHTTP(rec, req)

	// Should return 413 (Payload Too Large)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d for oversized request, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestSizeLimitPlugin_RequestBodyWithinLimit(t *testing.T) {
	// Create a mock next handler that reads the request body
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Received: " + string(body)))
	})

	// Create the plugin with a 1000-byte request limit
	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body":  1000,
		"max_response_body": 1000,
	})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	// Create a request with body smaller than 1000 bytes
	smallBody := "test data"
	req := httptest.NewRequest("POST", "/test", strings.NewReader(smallBody))
	rec := httptest.NewRecorder()

	// Execute the middleware
	mw(handler).ServeHTTP(rec, req)

	// Should return 200 OK
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for request within limit, got %d", http.StatusOK, rec.Code)
	}

	// Check response body
	if !strings.Contains(rec.Body.String(), smallBody) {
		t.Errorf("expected response to contain request body")
	}
}

func TestSizeLimitPlugin_ResponseBodyExceedsLimit(t *testing.T) {
	// Create a mock next handler that writes a large response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Try to write 200 bytes
		w.Write([]byte(strings.Repeat("b", 200)))
	})

	// Create the plugin with a 100-byte response limit
	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body":  1000,
		"max_response_body": 100,
	})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute the middleware
	mw(handler).ServeHTTP(rec, req)

	// Should return 413 (Payload Too Large) when the response size limit is exceeded
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d for oversized response, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestSizeLimitPlugin_ResponseBodyWithinLimit(t *testing.T) {
	// Create a mock next handler that writes a response
	responseData := "This is a test response"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseData))
	})

	// Create the plugin with a 1000-byte response limit
	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body":  1000,
		"max_response_body": 1000,
	})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute the middleware
	mw(handler).ServeHTTP(rec, req)

	// Should return 200 OK
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for response within limit, got %d", http.StatusOK, rec.Code)
	}

	// Check response body
	if rec.Body.String() != responseData {
		t.Errorf("expected response '%s', got '%s'", responseData, rec.Body.String())
	}
}

func TestSizeLimitPlugin_DefaultConfiguration(t *testing.T) {
	// Create the plugin with no configuration (should use defaults)
	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to create plugin with default config: %v", err)
	}

	// Create a small request
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	// Should work fine with defaults
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d with default config, got %d", http.StatusOK, rec.Code)
	}
}

func TestSizeLimitPlugin_InvalidConfiguration_NegativeRequestLimit(t *testing.T) {
	// Try to create the plugin with negative request limit
	_, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body": -100,
	})
	if err == nil {
		t.Error("expected error for negative max_request_body, got nil")
	}
}

func TestSizeLimitPlugin_InvalidConfiguration_NegativeResponseLimit(t *testing.T) {
	// Try to create the plugin with negative response limit
	_, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_response_body": -100,
	})
	if err == nil {
		t.Error("expected error for negative max_response_body, got nil")
	}
}

func TestSizeLimitPlugin_InvalidConfiguration_ZeroRequestLimit(t *testing.T) {
	// Try to create the plugin with zero request limit
	_, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body": 0,
	})
	if err == nil {
		t.Error("expected error for zero max_request_body, got nil")
	}
}

func TestSizeLimitPlugin_InvalidConfiguration_ZeroResponseLimit(t *testing.T) {
	// Try to create the plugin with zero response limit
	_, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_response_body": 0,
	})
	if err == nil {
		t.Error("expected error for zero max_response_body, got nil")
	}
}

func TestSizeLimitPlugin_InvalidConfiguration_WrongType(t *testing.T) {
	// Try to create the plugin with wrong type for request limit
	_, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body": "not-a-number",
	})
	if err == nil {
		t.Error("expected error for non-numeric max_request_body, got nil")
	}

	// Try with wrong type for response limit
	_, err = builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_response_body": "not-a-number",
	})
	if err == nil {
		t.Error("expected error for non-numeric max_response_body, got nil")
	}
}

func TestSizeLimitPlugin_EdgeCase_ExactLimit(t *testing.T) {
	// Test request body at exact limit
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Received: " + string(body)))
	})

	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body":  100,
		"max_response_body": 1000,
	})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	// Create a request with body exactly 100 bytes
	exactBody := strings.Repeat("a", 100)
	req := httptest.NewRequest("POST", "/test", strings.NewReader(exactBody))
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	// Should succeed at exact limit
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for request at exact limit, got %d", http.StatusOK, rec.Code)
	}
}

func TestSizeLimitPlugin_EdgeCase_OneByteBeyondLimit(t *testing.T) {
	// Test request body one byte beyond limit
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body":  100,
		"max_response_body": 1000,
	})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	// Create a request with body exactly 101 bytes (1 byte over limit)
	beyondBody := strings.Repeat("a", 101)
	req := httptest.NewRequest("POST", "/test", strings.NewReader(beyondBody))
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	// Should return 413
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d for request one byte beyond limit, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestSizeLimitPlugin_MultipleWrites(t *testing.T) {
	// Test response with multiple writes that collectively exceed limit
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write in chunks
		for i := 0; i < 5; i++ {
			w.Write([]byte(strings.Repeat("x", 30)))
		}
	})

	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body":  1000,
		"max_response_body": 100,
	})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	// After some writes succeed, status is already 200 and can't be changed
	// The important thing is that not all data was written (only first 90 bytes)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 (headers already sent), got %d", rec.Code)
	}

	// Verify that we didn't write all 150 bytes (5 * 30), only up to the limit
	if rec.Body.Len() > 100 {
		t.Errorf("expected body length <= 100 bytes, got %d", rec.Body.Len())
	}
}

func TestSizeLimitPlugin_EmptyBody(t *testing.T) {
	// Test with empty request body
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body":  1000,
		"max_response_body": 1000,
	})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	// Should work fine with empty body
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for empty body, got %d", http.StatusOK, rec.Code)
	}
}

func TestSizeLimitPlugin_LargeRequestBody(t *testing.T) {
	// Test with a very large request body to simulate real-world attack
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read all
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body":  1024, // 1KB limit
		"max_response_body": 1024,
	})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	// Create a 10KB request body
	largeBody := bytes.Repeat([]byte("a"), 10*1024)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(largeBody))
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	// Should return 413
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d for large request body, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestSizeLimitPlugin_Float64Configuration(t *testing.T) {
	// Test that float64 configuration values are handled correctly
	// (YAML parsers may interpret large numbers as float64)
	mw, err := builtins["size_limit"]("test-size-limit", map[string]interface{}{
		"max_request_body":  float64(10485760), // 10MB as float64
		"max_response_body": float64(52428800), // 50MB as float64
	})
	if err != nil {
		t.Fatalf("failed to create plugin with float64 config: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	// Should work fine
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d with float64 config, got %d", http.StatusOK, rec.Code)
	}
}
