package plugins

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDPlugin(t *testing.T) {
	var receivedRequestID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequestID = r.Header.Get("X-Request-ID")
		if receivedRequestID == "" {
			t.Error("expected X-Request-ID header in request")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Get registered plugin
	factory := builtins["request-id"]
	if factory == nil {
		t.Fatal("request-id plugin not registered")
	}

	// Create middleware (no config needed)
	mw, err := factory("request-id", nil)
	if err != nil {
		t.Fatalf("failed to create plugin middleware: %v", err)
	}

	req := httptest.NewRequest("GET", "/test-path", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	responseRequestID := rec.Header().Get("X-Request-ID")
	if responseRequestID == "" {
		t.Error("expected X-Request-ID header in response")
	}

	if responseRequestID != receivedRequestID {
		t.Errorf("request and response IDs don't match")
	}

	// Check it's a valid hex string (32 chars for 16 bytes)
	if len(responseRequestID) != 32 {
		t.Errorf("expected ID length 32, got %d", len(responseRequestID))
	}
}
