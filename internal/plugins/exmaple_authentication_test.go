package plugins

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticationPlugin(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		configAPIKey   string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid API Key",
			apiKey:         "test-secret-key",
			configAPIKey:   "test-secret-key",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Missing API Key",
			apiKey:         "",
			configAPIKey:   "test-secret-key",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized\n",
		},
		{
			name:           "Invalid API Key",
			apiKey:         "wrong-key",
			configAPIKey:   "test-secret-key",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Create a mock backend handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// 2. Get the registered plugin factory from builtins map
			factory := builtins["custom-auth"]
			if factory == nil {
				t.Fatal("custom-auth plugin not registered")
			}

			// 3. Create the middleware with test config
			mw, err := factory("custom-auth", map[string]interface{}{
				"apiKey": tt.configAPIKey,
			})
			if err != nil {
				t.Fatalf("failed to create plugin middleware: %v", err)
			}

			// 4. Create test request
			req := httptest.NewRequest("GET", "/test-path", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			// 5. Record the response
			rec := httptest.NewRecorder()

			// 6. Execute the middleware
			mw(handler).ServeHTTP(rec, req)

			// 7. Assert the results
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
			if rec.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rec.Body.String())
			}
		})
	}
}
