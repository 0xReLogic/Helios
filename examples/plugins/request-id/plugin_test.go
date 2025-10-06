package requestid

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
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

	mw, err := RegisterBuiltin("request-id", func(name string, cfg map[string]interface{}) (Middleware, error) {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id := rand.Int63()
				idStr := strconv.FormatInt(id, 10)

				r.Header.Set("X-Request-ID", idStr)

				w.Header().Set("X-Request-ID", idStr)

				next.ServeHTTP(w, r)
			})
		}, nil
	})("test-request-id", nil)
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
		t.Errorf("expected X-Request-ID in response '%s' to match received request ID '%s', but they did not", responseRequestID, receivedRequestID)
	}
}
