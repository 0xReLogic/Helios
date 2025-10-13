package plugins

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// largeBody is a string larger than typical min_size for compression tests.
const largeBody = `{"message": "This is a very large JSON response that should be compressed. It needs to be long enough to exceed any reasonable minimum size threshold for gzip compression. We will repeat this string multiple times to ensure it's sufficiently large. This is a very large JSON response that should be compressed. It needs to be long enough to exceed any reasonable minimum size threshold for gzip compression. We will repeat this string multiple times to ensure it's sufficiently large. This is a very large JSON response that should be compressed. It needs to be long enough to exceed any reasonable minimum size threshold for gzip compression. We will repeat this string multiple times to ensure it's sufficiently large. This is a very large JSON response that should be compressed. It needs to be long enough to exceed any reasonable minimum size threshold for gzip compression. We will repeat this string multiple times to ensure it's sufficiently large. This is a very large JSON response that should be compressed. It needs to be long enough to exceed any reasonable minimum size threshold for gzip compression. We will repeat this string multiple times to ensure it's sufficiently large."}`

// smallBody is a string smaller than typical min_size for compression tests.
const smallBody = `{"message": "small"}`

func TestGzipCompression(t *testing.T) {
	tests := []struct {
		name               string
		handlerBody        string
		handlerType        string
		configLevel        int
		configMinSize      int
		configContentTypes []string
		acceptEncoding     string
		expectedStatus     int
		expectCompression  bool
		expectedBody       string // Only used if no compression is expected
	}{
		// A. Basic Compression
		{
			name:               "Basic Compression - Large JSON body",
			handlerBody:        largeBody,
			handlerType:        "application/json",
			configLevel:        gzip.DefaultCompression,
			configMinSize:      10, // Small min_size to ensure compression
			configContentTypes: []string{"application/json"},
			acceptEncoding:     "gzip",
			expectedStatus:     http.StatusOK,
			expectCompression:  true,
		},
		// B. Size Threshold Behavior
		{
			name:               "Size Threshold - Small JSON body",
			handlerBody:        smallBody,
			handlerType:        "application/json",
			configLevel:        gzip.DefaultCompression,
			configMinSize:      1024, // Large min_size to prevent compression
			configContentTypes: []string{"application/json"},
			acceptEncoding:     "gzip",
			expectedStatus:     http.StatusOK,
			expectCompression:  false,
			expectedBody:       smallBody,
		},
		// C. Content-Type Filtering - Case 1: JSON should compress
		{
			name:               "Content-Type Filtering - JSON (should compress)",
			handlerBody:        largeBody,
			handlerType:        "application/json",
			configLevel:        gzip.DefaultCompression,
			configMinSize:      10,
			configContentTypes: []string{"application/json"},
			acceptEncoding:     "gzip",
			expectedStatus:     http.StatusOK,
			expectCompression:  true,
		},
		// C. Content-Type Filtering - Case 2: Plain text should not compress
		{
			name:               "Content-Type Filtering - Plain Text (should not compress)",
			handlerBody:        largeBody,
			handlerType:        "text/plain",
			configLevel:        gzip.DefaultCompression,
			configMinSize:      10,
			configContentTypes: []string{"application/json"}, // Only JSON allowed
			acceptEncoding:     "gzip",
			expectedStatus:     http.StatusOK,
			expectCompression:  false,
			expectedBody:       largeBody,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Create a mock backend handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.handlerType)
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(tt.handlerBody)); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})

			// 2. Get the registered plugin factory from builtins map
			factory := builtins["gzip"]
			if factory == nil {
				t.Fatal("gzip plugin not registered")
			}

			// 3. Create the middleware with test config
			mw, err := factory("gzip", map[string]interface{}{
				"level":         float64(tt.configLevel),
				"min_size":      float64(tt.configMinSize),
				"content_types": convertStringsToInterfaces(tt.configContentTypes),
			})
			if err != nil {
				t.Fatalf("failed to create plugin middleware: %v", err)
			}

			// 4. Create test request
			req := httptest.NewRequest("GET", "/test-path", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			// 5. Record the response
			rec := httptest.NewRecorder()

			// 6. Execute the middleware
			mw(handler).ServeHTTP(rec, req)

			// 7. Assert the results
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectCompression {
				// Assert: Response header Content-Encoding: gzip exists.
				if rec.Header().Get("Content-Encoding") != "gzip" {
					t.Errorf("expected Content-Encoding: gzip header, got %q", rec.Header().Get("Content-Encoding"))
				}

				// Assert: Body is smaller than original.
				if len(rec.Body.Bytes()) >= len([]byte(tt.handlerBody)) {
					t.Errorf("expected compressed body length (%d) to be smaller than original (%d)", len(rec.Body.Bytes()), len([]byte(tt.handlerBody)))
				}

				// Assert: Decompressing the body yields the original content.
				gzr, err := gzip.NewReader(rec.Body)
				if err != nil {
					t.Fatalf("failed to create gzip reader: %v", err)
				}
				defer gzr.Close()

				decompressedBody, err := io.ReadAll(gzr)
				if err != nil {
					t.Fatalf("failed to decompress body: %v", err)
				}

				if string(decompressedBody) != tt.handlerBody {
					t.Errorf("decompressed body mismatch: expected %q, got %q", tt.handlerBody, string(decompressedBody))
				}
			} else {
				// Assert: No Content-Encoding header.
				if rec.Header().Get("Content-Encoding") != "" {
					t.Errorf("expected no Content-Encoding header, got %q", rec.Header().Get("Content-Encoding"))
				}

				// Assert: Body is uncompressed (identical to original).
				if rec.Body.String() != tt.expectedBody {
					t.Errorf("expected body %q, got %q", tt.expectedBody, rec.Body.String())
				}
			}
		})
	}
}

// Helper to convert []string to []interface{} for plugin config
func convertStringsToInterfaces(s []string) []interface{} {
	if s == nil {
		return nil
	}
	interfaces := make([]interface{}, len(s))
	for i, v := range s {
		interfaces[i] = v
	}
	return interfaces
}
