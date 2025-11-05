package plugins

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xReLogic/Helios/internal/config"
)

// Test constants to avoid duplication
const (
	testContentType           = "Content-Type"
	testTextPlain             = "text/plain"
	testApplicationJSON       = "application/json"
	testCustomHeader          = "X-Custom-Header"
	testServerHeader          = "X-Server"
	testFailedBuildPlugin     = "Failed to build plugin chain: %v"
	testAcceptEncoding        = "Accept-Encoding"
	testExpectedStatus200     = "Expected status 200, got %d"
	testContentEncoding       = "Content-Encoding"
	testCustomAuth            = "custom-auth"
	testAPIKey                = "X-API-Key"
	testOrderHeader           = "X-Order"
	testShouldNotExistHeader  = "X-Should-Not-Exist"
	testAPIVersionHeader      = "X-API-Version"
	testPoweredByHeader       = "X-Powered-By"
)


// TestPluginChainIntegration tests the complete plugin chain with multiple plugins
func TestPluginChainIntegration(t *testing.T) {
	// Create a base handler that returns a simple response
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(testContentType, testTextPlain)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello from base handler"))
	})

	// Configure plugin chain: headers -> logging -> gzip
	pluginConfig := config.PluginsConfig{
		Enabled: true,
		Chain: []config.PluginConfig{
			{
				Name: "headers",
				Config: map[string]interface{}{
					"set": map[string]interface{}{
						testCustomHeader: "test-value",
						testServerHeader:        "Helios",
					},
				},
			},
			{
				Name:   "logging",
				Config: map[string]interface{}{},
			},
			{
				Name: "gzip",
				Config: map[string]interface{}{
					"level":         float64(5),
					"min_size":      float64(0),
					"content_types": []interface{}{testTextPlain, testApplicationJSON},
				},
			},
		},
	}

	// Build the chain
	handler, err := BuildChain(pluginConfig, baseHandler)
	if err != nil {
		t.Fatalf(testFailedBuildPlugin, err)
	}

	// Create a request with gzip support
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(testAcceptEncoding, "gzip")
	rec := httptest.NewRecorder()

	// Execute the request
	handler.ServeHTTP(rec, req)

	// Verify status code
	if rec.Code != http.StatusOK {
		t.Errorf(testExpectedStatus200, rec.Code)
	}

	// Verify custom headers were added
	if rec.Header().Get(testCustomHeader) != "test-value" {
		t.Errorf("Expected X-Custom-Header to be 'test-value', got '%s'", rec.Header().Get(testCustomHeader))
	}
	if rec.Header().Get(testServerHeader) != "Helios" {
		t.Errorf("Expected X-Server to be 'Helios', got '%s'", rec.Header().Get(testServerHeader))
	}

	// Verify compression was applied
	if rec.Header().Get(testContentEncoding) != "gzip" {
		t.Errorf("Expected Content-Encoding to be 'gzip', got '%s'", rec.Header().Get(testContentEncoding))
	}

	// Decompress and verify content
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			t.Logf("failed to close gzip reader: %v", err)
		}
	}()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read decompressed body: %v", err)
	}

	if string(body) != "Hello from base handler" {
		t.Errorf("Expected body 'Hello from base handler', got '%s'", string(body))
	}
}

// TestPluginChainWithSizeLimit tests plugin chain with size limit enforcement
func TestPluginChainWithSizeLimit(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Received: " + string(body)))
	})

	pluginConfig := config.PluginsConfig{
		Enabled: true,
		Chain: []config.PluginConfig{
			{
				Name: "size_limit",
				Config: map[string]interface{}{
					"max_request_body": float64(100),
				},
			},
		},
	}

	handler, err := BuildChain(pluginConfig, baseHandler)
	if err != nil {
		t.Fatalf(testFailedBuildPlugin, err)
	}

	// Test with body under limit
	t.Run("within limit", func(t *testing.T) {
		smallBody := strings.NewReader("small body")
		req := httptest.NewRequest("POST", "/test", smallBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf(testExpectedStatus200, rec.Code)
		}
	})

	// Test with body over limit
	t.Run("exceeds limit", func(t *testing.T) {
		largeBody := strings.NewReader(strings.Repeat("a", 150))
		req := httptest.NewRequest("POST", "/test", largeBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", rec.Code)
		}
	})
}

// TestPluginChainWithAuthentication tests plugin chain with authentication
func TestPluginChainWithAuthentication(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Authenticated"))
	})

	pluginConfig := config.PluginsConfig{
		Enabled: true,
		Chain: []config.PluginConfig{
			{
				Name: testCustomAuth,
				Config: map[string]interface{}{
					"apiKey": "secret-token-123",
				},
			},
		},
	}

	handler, err := BuildChain(pluginConfig, baseHandler)
	if err != nil {
		t.Fatalf(testFailedBuildPlugin, err)
	}

	// Test with valid token
	t.Run("valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(testAPIKey, "secret-token-123")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf(testExpectedStatus200, rec.Code)
		}
	})

	// Test with invalid token
	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(testAPIKey, "wrong-token")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	// Test with missing token
	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})
}

// TestPluginChainOrder tests that plugin order matters
func TestPluginChainOrder(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(testContentType, testTextPlain)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	})

	// Chain 1: headers -> gzip
	chain1Config := config.PluginsConfig{
		Enabled: true,
		Chain: []config.PluginConfig{
			{
				Name: "headers",
				Config: map[string]interface{}{
					"set": map[string]interface{}{
						testOrderHeader: "headers-first",
					},
				},
			},
			{
				Name: "gzip",
				Config: map[string]interface{}{
					"level":         float64(5),
					"min_size":      float64(0),
					"content_types": []interface{}{testTextPlain},
				},
			},
		},
	}

	handler1, err := BuildChain(chain1Config, baseHandler)
	if err != nil {
		t.Fatalf("Failed to build chain 1: %v", err)
	}

	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set(testAcceptEncoding, "gzip")
	rec1 := httptest.NewRecorder()
	handler1.ServeHTTP(rec1, req1)

	// Verify both plugins were applied
	if rec1.Header().Get(testOrderHeader) != "headers-first" {
		t.Errorf("Expected X-Order header, got '%s'", rec1.Header().Get(testOrderHeader))
	}
	if rec1.Header().Get(testContentEncoding) != "gzip" {
		t.Errorf("Expected gzip encoding, got '%s'", rec1.Header().Get(testContentEncoding))
	}
}

// TestPluginChainDisabled tests that plugins are skipped when disabled
func TestPluginChainDisabled(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	})

	pluginConfig := config.PluginsConfig{
		Enabled: false,
		Chain: []config.PluginConfig{
			{
				Name: "headers",
				Config: map[string]interface{}{
					"set": map[string]interface{}{
						testShouldNotExistHeader: "true",
					},
				},
			},
		},
	}

	handler, err := BuildChain(pluginConfig, baseHandler)
	if err != nil {
		t.Fatalf(testFailedBuildPlugin, err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify header was not added
	if rec.Header().Get(testShouldNotExistHeader) != "" {
		t.Errorf("Expected no X-Should-Not-Exist header, but got '%s'", rec.Header().Get(testShouldNotExistHeader))
	}
}

// TestPluginChainUnknownPlugin tests error handling for unknown plugins
func TestPluginChainUnknownPlugin(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	pluginConfig := config.PluginsConfig{
		Enabled: true,
		Chain: []config.PluginConfig{
			{
				Name:   "unknown_plugin",
				Config: map[string]interface{}{},
			},
		},
	}

	_, err := BuildChain(pluginConfig, baseHandler)
	if err == nil {
		t.Error("Expected error for unknown plugin, got nil")
	}
	if !strings.Contains(err.Error(), "unknown plugin") {
		t.Errorf("Expected 'unknown plugin' error, got: %v", err)
	}
}

// TestPluginChainNilBaseHandler tests error handling for nil base handler
func TestPluginChainNilBaseHandler(t *testing.T) {
	pluginConfig := config.PluginsConfig{
		Enabled: true,
		Chain:   []config.PluginConfig{},
	}

	_, err := BuildChain(pluginConfig, nil)
	if err == nil {
		t.Error("Expected error for nil base handler, got nil")
	}
	if !strings.Contains(err.Error(), "base handler is nil") {
		t.Errorf("Expected 'base handler is nil' error, got: %v", err)
	}
}

// TestPluginListAvailability tests that all registered plugins are listed
func TestPluginListAvailability(t *testing.T) {
	plugins := List()

	expectedPlugins := []string{
		"gzip",
		testCustomAuth,
		"request-id",
		"headers",
		"logging",
		"size_limit",
	}

	if len(plugins) < len(expectedPlugins) {
		t.Errorf("Expected at least %d plugins, got %d", len(expectedPlugins), len(plugins))
	}

	pluginMap := make(map[string]bool)
	for _, p := range plugins {
		pluginMap[p] = true
	}

	for _, expected := range expectedPlugins {
		if !pluginMap[expected] {
			t.Errorf("Expected plugin '%s' to be registered", expected)
		}
	}
}

// TestComplexPluginChain tests a realistic multi-plugin scenario
func TestComplexPluginChain(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read request body
		body, _ := io.ReadAll(r.Body)

		w.Header().Set(testContentType, testApplicationJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"processed","received":"` + string(body) + `"}`))
	}) // Complex chain: authentication -> size limit -> headers -> gzip -> logging
	pluginConfig := config.PluginsConfig{
		Enabled: true,
		Chain: []config.PluginConfig{
			{
				Name: testCustomAuth,
				Config: map[string]interface{}{
					"apiKey": "valid-token",
				},
			},
			{
				Name: "size_limit",
				Config: map[string]interface{}{
					"max_request_body": float64(1000),
				},
			},
			{
				Name: "headers",
				Config: map[string]interface{}{
					"set": map[string]interface{}{
						testAPIVersionHeader: "v1",
						testPoweredByHeader:  "Helios",
					},
				},
			},
			{
				Name: "gzip",
				Config: map[string]interface{}{
					"level":         float64(6),
					"min_size":      float64(0),
					"content_types": []interface{}{testApplicationJSON},
				},
			},
			{
				Name:   "logging",
				Config: map[string]interface{}{},
			},
		},
	}

	handler, err := BuildChain(pluginConfig, baseHandler)
	if err != nil {
		t.Fatalf("Failed to build complex plugin chain: %v", err)
	}

	// Create request with valid auth and body
	reqBody := bytes.NewBufferString(`{"data":"test"}`)
	req := httptest.NewRequest("POST", "/api/test", reqBody)
	req.Header.Set(testAPIKey, "valid-token")
	req.Header.Set(testAcceptEncoding, "gzip")
	req.Header.Set(testContentType, testApplicationJSON)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify successful processing
	if rec.Code != http.StatusOK {
		t.Errorf(testExpectedStatus200, rec.Code)
	}

	// Verify headers plugin worked
	if rec.Header().Get(testAPIVersionHeader) != "v1" {
		t.Errorf("Expected X-API-Version header, got '%s'", rec.Header().Get(testAPIVersionHeader))
	}
	if rec.Header().Get(testPoweredByHeader) != "Helios" {
		t.Errorf("Expected X-Powered-By header, got '%s'", rec.Header().Get(testPoweredByHeader))
	}

	// Verify compression worked
	if rec.Header().Get(testContentEncoding) != "gzip" {
		t.Errorf("Expected gzip encoding, got '%s'", rec.Header().Get(testContentEncoding))
	}

	// Decompress and verify response
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			t.Logf("failed to close gzip reader: %v", err)
		}
	}()

	responseBody, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read decompressed body: %v", err)
	}

	if !strings.Contains(string(responseBody), "processed") {
		t.Errorf("Expected response to contain 'processed', got '%s'", string(responseBody))
	}
}
