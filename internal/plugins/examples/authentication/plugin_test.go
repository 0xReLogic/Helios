package authentication

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type Middleware func(next http.Handler) http.Handler
type factory func(name string, cfg map[string]interface{}) (Middleware, error)

func getAuthenticationMiddleware(name string, cfg map[string]interface{}) (Middleware, error) {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "super-secret-key" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

func TestAuthenticationPlugin(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid API Key",
			apiKey:         "super-secret-key",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Missing API Key",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized\n",
		},
		{
			name:           "Invalid API Key",
			apiKey:         "wrong-key",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			mw, err := getAuthenticationMiddleware("custom-auth", nil)
			if err != nil {
				t.Fatalf("failed to create plugin middleware: %v", err)
			}

			req := httptest.NewRequest("GET", "/test-path", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			rec := httptest.NewRecorder()

			mw(handler).ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
			if rec.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rec.Body.String())
			}
		})
	}
}
