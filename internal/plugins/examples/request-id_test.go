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
			t.Error("expected X-Request-ID header in downstream request, but got empty")
		}
		w.WriteHeader(http.StatusOK)
	})

	factory := plugins.builtins["request-id"]
	if factory == nil {
		t.Fatal("request-id plugin not registered")
	}

	mw, err := factory("request-id", nil)
	if err != nil {
		t.Fatalf("failed to create plugin middleware: %v", err)
	}

	req := httptest.NewRequest("GET", "/test-path", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	responseRequestID := rec.Header().Get("X-Request-ID")
	if responseRequestID == "" {
		t.Error("expected X-Request-ID header in response, but got empty")
	}

	if responseRequestID != receivedRequestID {
		t.Errorf("expected X-Request-ID in response '%s' to match received request ID '%s'", responseRequestID, receivedRequestID)
	}

	if len(responseRequestID) != 32 {
		t.Errorf("expected request ID length 32, got %d", len(responseRequestID))
	}
}
