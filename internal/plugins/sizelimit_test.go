package plugins

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	// Test constants to reduce code duplication
	testPluginName          = "size_limit"
	testCreateErr           = "failed to create plugin: %v"
	testPath                = "/test"
	testMaxRequestBodyKey   = "max_request_body"
	testMaxResponseBodyKey  = "max_response_body"
	testExpectedStatusErr   = "expected status %d, got %d"
	testExpectedBodyLenErr  = "expected body length exactly %d bytes (3 successful writes), got %d"
	testExpectedBodyContain = "expected response to contain '%s'"
)

// Helper functions to reduce test code duplication

// createSizeLimitPlugin creates a size limit plugin with the given limits
func createSizeLimitPlugin(t *testing.T, maxRequestBody, maxResponseBody int) Middleware {
	t.Helper()
	mw, err := builtins[testPluginName](testPluginName, map[string]interface{}{
		testMaxRequestBodyKey:  maxRequestBody,
		testMaxResponseBodyKey: maxResponseBody,
	})
	if err != nil {
		t.Fatalf(testCreateErr, err)
	}
	return mw
}

// testRequest holds parameters for executing a test request
type testRequest struct {
	middleware Middleware
	handler    http.Handler
	method     string
	body       io.Reader
}

// executeRequest executes the middleware with the given test request parameters
func executeRequest(req testRequest) *httptest.ResponseRecorder {
	httpReq := httptest.NewRequest(req.method, testPath, req.body)
	rec := httptest.NewRecorder()
	req.middleware(req.handler).ServeHTTP(rec, httpReq)
	return rec
}

// assertStatusCode checks that the response has the expected status code
func assertStatusCode(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rec.Code != expected {
		t.Errorf(testExpectedStatusErr, expected, rec.Code)
	}
}

// simpleOKHandler returns a handler that writes "ok"
func simpleOKHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok")) // Explicitly ignore in tests
	})
}

func TestSizeLimitPlugin_RequestBodyLimits(t *testing.T) {
	tests := []struct {
		name               string
		bodySize           int
		limit              int
		expectedStatus     int
		expectBodyContains string
	}{
		{
			name:           "Request body exceeds limit",
			bodySize:       200,
			limit:          100,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:               "Request body within limit",
			bodySize:           50,
			limit:              1000,
			expectedStatus:     http.StatusOK,
			expectBodyContains: "Received:",
		},
		{
			name:           "Request body at exact limit",
			bodySize:       100,
			limit:          100,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Request body one byte beyond limit",
			bodySize:       101,
			limit:          100,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "Large request body (DoS simulation)",
			bodySize:       10 * 1024, // 10KB
			limit:          1024,      // 1KB
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler that reads request body
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("Received: " + string(body))) // Explicitly ignore in tests
			})

			// Create plugin with specified limit
			mw := createSizeLimitPlugin(t, tt.limit, 10000)

			// Create request with specified body size and execute middleware
			requestBody := bytes.Repeat([]byte("a"), tt.bodySize)
			rec := executeRequest(testRequest{
				middleware: mw,
				handler:    handler,
				method:     "POST",
				body:       bytes.NewReader(requestBody),
			})

			// Assert status code
			assertStatusCode(t, rec, tt.expectedStatus)

			// Assert response body if specified
			if tt.expectBodyContains != "" && !strings.Contains(rec.Body.String(), tt.expectBodyContains) {
				t.Errorf(testExpectedBodyContain, tt.expectBodyContains)
			}
		})
	}
}

func TestSizeLimitPlugin_ResponseBodyExceedsLimit(t *testing.T) {
	// Create a mock next handler that writes a large response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("b", 200))) // Explicitly ignore in tests - Try to write 200 bytes
	})

	mw := createSizeLimitPlugin(t, 1000, 100) // 100-byte response limit
	rec := executeRequest(testRequest{
		middleware: mw,
		handler:    handler,
		method:     "GET",
	})

	// Should return 413 when the response size limit is exceeded
	assertStatusCode(t, rec, http.StatusRequestEntityTooLarge)
}

func TestSizeLimitPlugin_ResponseBodyWithinLimit(t *testing.T) {
	responseData := "This is a test response"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseData)) // Explicitly ignore in tests
	})

	mw := createSizeLimitPlugin(t, 1000, 1000)
	rec := executeRequest(testRequest{
		middleware: mw,
		handler:    handler,
		method:     "GET",
	})

	assertStatusCode(t, rec, http.StatusOK)
	if rec.Body.String() != responseData {
		t.Errorf("expected response '%s', got '%s'", responseData, rec.Body.String())
	}
}

func TestSizeLimitPlugin_DefaultConfiguration(t *testing.T) {
	// Create the plugin with no configuration (should use defaults)
	mw, err := builtins[testPluginName](testPluginName, map[string]interface{}{})
	if err != nil {
		t.Fatalf(testCreateErr, err)
	}

	rec := executeRequest(testRequest{
		middleware: mw,
		handler:    simpleOKHandler(),
		method:     "GET",
	})
	assertStatusCode(t, rec, http.StatusOK)
}

func TestSizeLimitPlugin_InvalidConfiguration_NegativeRequestLimit(t *testing.T) {
	// Try to create the plugin with negative request limit
	_, err := builtins[testPluginName](testPluginName, map[string]interface{}{
		testMaxRequestBodyKey: -100,
	})
	if err == nil {
		t.Error("expected error for negative max_request_body, got nil")
	}
}

func TestSizeLimitPlugin_InvalidConfiguration_NegativeResponseLimit(t *testing.T) {
	// Try to create the plugin with negative response limit
	_, err := builtins[testPluginName](testPluginName, map[string]interface{}{
		testMaxResponseBodyKey: -100,
	})
	if err == nil {
		t.Error("expected error for negative max_response_body, got nil")
	}
}

func TestSizeLimitPlugin_InvalidConfiguration_ZeroRequestLimit(t *testing.T) {
	// Try to create the plugin with zero request limit
	_, err := builtins[testPluginName](testPluginName, map[string]interface{}{
		testMaxRequestBodyKey: 0,
	})
	if err == nil {
		t.Error("expected error for zero max_request_body, got nil")
	}
}

func TestSizeLimitPlugin_InvalidConfiguration_ZeroResponseLimit(t *testing.T) {
	// Try to create the plugin with zero response limit
	_, err := builtins[testPluginName](testPluginName, map[string]interface{}{
		testMaxResponseBodyKey: 0,
	})
	if err == nil {
		t.Error("expected error for zero max_response_body, got nil")
	}
}

func TestSizeLimitPlugin_InvalidConfiguration_WrongType(t *testing.T) {
	// Try to create the plugin with wrong type for request limit
	_, err := builtins[testPluginName](testPluginName, map[string]interface{}{
		testMaxRequestBodyKey: "not-a-number",
	})
	if err == nil {
		t.Error("expected error for non-numeric max_request_body, got nil")
	}

	// Try with wrong type for response limit
	_, err = builtins[testPluginName](testPluginName, map[string]interface{}{
		testMaxResponseBodyKey: "not-a-number",
	})
	if err == nil {
		t.Error("expected error for non-numeric max_response_body, got nil")
	}
}

func TestSizeLimitPlugin_MultipleWrites(t *testing.T) {
	// Test response with multiple writes that collectively exceed limit
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write in chunks: 5 * 30 bytes = 150 bytes total
		for i := 0; i < 5; i++ {
			_, _ = w.Write([]byte(strings.Repeat("x", 30))) // Explicitly ignore in tests
		}
	})

	mw := createSizeLimitPlugin(t, 1000, 100)
	rec := executeRequest(testRequest{
		middleware: mw,
		handler:    handler,
		method:     "GET",
	})

	// After some writes succeed, status is already 200 and can't be changed
	assertStatusCode(t, rec, http.StatusOK)

	// Verify exact truncation: 3 writes of 30 bytes = 90 bytes (4th write fails at 120 > 100)
	expectedBodyLen := 90
	if rec.Body.Len() != expectedBodyLen {
		t.Errorf(testExpectedBodyLenErr, expectedBodyLen, rec.Body.Len())
	}
}

func TestSizeLimitPlugin_EmptyBody(t *testing.T) {
	mw := createSizeLimitPlugin(t, 1000, 1000)
	rec := executeRequest(testRequest{
		middleware: mw,
		handler:    simpleOKHandler(),
		method:     "GET",
	})
	assertStatusCode(t, rec, http.StatusOK)
}

func TestSizeLimitPlugin_Float64Configuration(t *testing.T) {
	// Test that float64 configuration values are handled correctly
	// (YAML parsers may interpret large numbers as float64)
	mw, err := builtins[testPluginName](testPluginName, map[string]interface{}{
		testMaxRequestBodyKey:  float64(10485760), // 10MB as float64
		testMaxResponseBodyKey: float64(52428800), // 50MB as float64
	})
	if err != nil {
		t.Fatalf(testCreateErr, err)
	}

	rec := executeRequest(testRequest{
		middleware: mw,
		handler:    simpleOKHandler(),
		method:     "GET",
	})
	assertStatusCode(t, rec, http.StatusOK)
}
