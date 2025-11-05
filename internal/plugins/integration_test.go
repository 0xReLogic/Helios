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

// TestPluginChainIntegration tests the complete plugin chain with multiple plugins
func TestPluginChainIntegration(t *testing.T) {
	// Create a base handler that returns a simple response
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from base handler"))
	})

	// Configure plugin chain: headers -> logging -> gzip
	pluginConfig := config.PluginsConfig{
		Enabled: true,
		Chain: []config.PluginConfig{
			{
				Name: "headers",
				Config: map[string]interface{}{
					"set": map[string]interface{}{
						"X-Custom-Header": "test-value",
						"X-Server":        "Helios",
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
					"content_types": []interface{}{"text/plain", "application/json"},
				},
			},
		},
	}

	// Build the chain
	handler, err := BuildChain(pluginConfig, baseHandler)
	if err != nil {
		t.Fatalf("Failed to build plugin chain: %v", err)
	}

	// Create a request with gzip support
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	// Execute the request
	handler.ServeHTTP(rec, req)

	// Verify status code
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify custom headers were added
	if rec.Header().Get("X-Custom-Header") != "test-value" {
		t.Errorf("Expected X-Custom-Header to be 'test-value', got '%s'", rec.Header().Get("X-Custom-Header"))
	}
	if rec.Header().Get("X-Server") != "Helios" {
		t.Errorf("Expected X-Server to be 'Helios', got '%s'", rec.Header().Get("X-Server"))
	}

	// Verify compression was applied
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Expected Content-Encoding to be 'gzip', got '%s'", rec.Header().Get("Content-Encoding"))
	}

	// Decompress and verify content
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()

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
		w.Write([]byte("Received: " + string(body)))
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
		t.Fatalf("Failed to build plugin chain: %v", err)
	}

	// Test with body under limit
	t.Run("within limit", func(t *testing.T) {
		smallBody := strings.NewReader("small body")
		req := httptest.NewRequest("POST", "/test", smallBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
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
		w.Write([]byte("Authenticated"))
	})

	pluginConfig := config.PluginsConfig{
		Enabled: true,
		Chain: []config.PluginConfig{
			{
				Name: "custom-auth",
				Config: map[string]interface{}{
					"apiKey": "secret-token-123",
				},
			},
		},
	}

	handler, err := BuildChain(pluginConfig, baseHandler)
	if err != nil {
		t.Fatalf("Failed to build plugin chain: %v", err)
	}

	// Test with valid token
	t.Run("valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "secret-token-123")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	// Test with invalid token
	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "wrong-token")
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
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	// Chain 1: headers -> gzip
	chain1Config := config.PluginsConfig{
		Enabled: true,
		Chain: []config.PluginConfig{
			{
				Name: "headers",
				Config: map[string]interface{}{
					"set": map[string]interface{}{
						"X-Order": "headers-first",
					},
				},
			},
			{
				Name: "gzip",
				Config: map[string]interface{}{
					"level":         float64(5),
					"min_size":      float64(0),
					"content_types": []interface{}{"text/plain"},
				},
			},
		},
	}

	handler1, err := BuildChain(chain1Config, baseHandler)
	if err != nil {
		t.Fatalf("Failed to build chain 1: %v", err)
	}

	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("Accept-Encoding", "gzip")
	rec1 := httptest.NewRecorder()
	handler1.ServeHTTP(rec1, req1)

	// Verify both plugins were applied
	if rec1.Header().Get("X-Order") != "headers-first" {
		t.Errorf("Expected X-Order header, got '%s'", rec1.Header().Get("X-Order"))
	}
	if rec1.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Expected gzip encoding, got '%s'", rec1.Header().Get("Content-Encoding"))
	}
}

// TestPluginChainDisabled tests that plugins are skipped when disabled
func TestPluginChainDisabled(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	pluginConfig := config.PluginsConfig{
		Enabled: false,
		Chain: []config.PluginConfig{
			{
				Name: "headers",
				Config: map[string]interface{}{
					"set": map[string]interface{}{
						"X-Should-Not-Exist": "true",
					},
				},
			},
		},
	}

	handler, err := BuildChain(pluginConfig, baseHandler)
	if err != nil {
		t.Fatalf("Failed to build plugin chain: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify header was not added
	if rec.Header().Get("X-Should-Not-Exist") != "" {
		t.Errorf("Expected no X-Should-Not-Exist header, but got '%s'", rec.Header().Get("X-Should-Not-Exist"))
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
		"custom-auth",
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
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"processed","received":"` + string(body) + `"}`))
	})

	// Complex chain: authentication -> size limit -> headers -> gzip -> logging
	pluginConfig := config.PluginsConfig{
		Enabled: true,
		Chain: []config.PluginConfig{
			{
				Name: "custom-auth",
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
						"X-API-Version": "v1",
						"X-Powered-By":  "Helios",
					},
				},
			},
			{
				Name: "gzip",
				Config: map[string]interface{}{
					"level":         float64(6),
					"min_size":      float64(0),
					"content_types": []interface{}{"application/json"},
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
	req.Header.Set("X-API-Key", "valid-token")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify successful processing
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify headers plugin worked
	if rec.Header().Get("X-API-Version") != "v1" {
		t.Errorf("Expected X-API-Version header, got '%s'", rec.Header().Get("X-API-Version"))
	}
	if rec.Header().Get("X-Powered-By") != "Helios" {
		t.Errorf("Expected X-Powered-By header, got '%s'", rec.Header().Get("X-Powered-By"))
	}

	// Verify compression worked
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Expected gzip encoding, got '%s'", rec.Header().Get("Content-Encoding"))
	}

	// Decompress and verify response
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	responseBody, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read decompressed body: %v", err)
	}

	if !strings.Contains(string(responseBody), "processed") {
		t.Errorf("Expected response to contain 'processed', got '%s'", string(responseBody))
	}
}
