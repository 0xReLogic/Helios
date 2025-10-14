package plugins

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	ContentTypeJSON          = "application/json"
	FailedToWriteError       = "failed to write response: %v"
	PluginNotRegisteredError = "gzip plugin not registered"
	ExpectedStatusError      = "expected status %d, got %d"
)
const largeBody = `{"message": "This is a very large JSON response that should be compressed. It needs to be long enough to exceed any reasonable minimum size threshold for gzip compression. We will repeat this string multiple times to ensure it's sufficiently large. This is a very large JSON response that should be compressed. It needs to be long enough to exceed any reasonable minimum size threshold for gzip compression. We will repeat this string multiple times to ensure it's sufficiently large. This is a very large JSON response that should be compressed. It needs to be long enough to exceed any reasonable minimum size threshold for gzip compression. We will repeat this string multiple times to ensure it's sufficiently large. This is a very large JSON response that should be compressed. It needs to be long enough to exceed any reasonable minimum size threshold for gzip compression. We will repeat this string multiple times to ensure it's sufficiently large. This is a very large JSON response that should be compressed. It needs to be long enough to exceed any reasonable minimum size threshold for gzip compression. We will repeat this string multiple times to ensure it's sufficiently large."}`
const smallBody = `{"message": "small"}`

func newGzipMiddleware(t testing.TB, level, minSize int, contentTypes []string) Middleware {
	t.Helper()

	factory := builtins["gzip"]
	if factory == nil {
		t.Fatal(PluginNotRegisteredError)
	}

	mw, err := factory("gzip", map[string]interface{}{
		"level":         float64(level),
		"min_size":      float64(minSize),
		"content_types": convertStringsToInterfaces(contentTypes),
	})
	if err != nil {
		t.Fatalf("failed to create plugin middleware: %v", err)
	}

	return mw
}

func decompressBody(t *testing.T, data []byte) string {
	t.Helper()

	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	decompressedBody, err := io.ReadAll(gzr)
	if err != nil {
		t.Fatalf("failed to decompress body: %v", err)
	}

	return string(decompressedBody)
}

func assertCompressed(t *testing.T, rec *httptest.ResponseRecorder, expectedBody string) {
	// Assert: Response header Content-Encoding: gzip exists.
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected Content-Encoding: gzip header, got %q", rec.Header().Get("Content-Encoding"))
	}

	// Assert: Body is smaller than original.
	if len(rec.Body.Bytes()) >= len([]byte(expectedBody)) {
		t.Errorf("expected compressed body length (%d) to be smaller than original (%d)", len(rec.Body.Bytes()), len([]byte(expectedBody)))
	}

	// Assert: Decompressing the body yields the original content.
	decompressedBody := decompressBody(t, rec.Body.Bytes())

	if string(decompressedBody) != expectedBody {
		t.Errorf("decompressed body mismatch: expected %q, got %q", expectedBody, string(decompressedBody))
	}
}

func assertUncompressed(t *testing.T, rec *httptest.ResponseRecorder, expectedBody string) {
	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("expected no Content-Encoding header, got %q", rec.Header().Get("Content-Encoding"))
	}

	// Assert: Body is uncompressed (identical to original).
	if rec.Body.String() != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, rec.Body.String())
	}
}

func newMockHandler(t testing.TB, handlerType, handlerBody string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", handlerType)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(handlerBody)); err != nil {
			t.Fatalf(FailedToWriteError, err)
		}
	})
}

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
		expectedBody       string
	}{
		// A. Basic Compression
		{
			name:               "Basic Compression - Large JSON body",
			handlerBody:        largeBody,
			handlerType:        ContentTypeJSON,
			configLevel:        gzip.DefaultCompression,
			configMinSize:      10, // Small min_size to ensure compression
			configContentTypes: []string{ContentTypeJSON},
			acceptEncoding:     "gzip",
			expectedStatus:     http.StatusOK,
			expectCompression:  true,
		},
		// B. Size Threshold Behavior
		{
			name:               "Size Threshold - Small JSON body",
			handlerBody:        smallBody,
			handlerType:        ContentTypeJSON,
			configLevel:        gzip.DefaultCompression,
			configMinSize:      1024, // Large min_size to prevent compression
			configContentTypes: []string{ContentTypeJSON},
			acceptEncoding:     "gzip",
			expectedStatus:     http.StatusOK,
			expectCompression:  false,
			expectedBody:       smallBody,
		},
		// C. Content-Type Filtering - Case 1: JSON should compress
		{
			name:               "Content-Type Filtering - JSON (should compress)",
			handlerBody:        largeBody,
			handlerType:        ContentTypeJSON,
			configLevel:        gzip.DefaultCompression,
			configMinSize:      10,
			configContentTypes: []string{ContentTypeJSON},
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
			configContentTypes: []string{ContentTypeJSON}, // Only JSON allowed
			acceptEncoding:     "gzip",
			expectedStatus:     http.StatusOK,
			expectCompression:  false,
			expectedBody:       largeBody,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newMockHandler(t, tt.handlerType, tt.handlerBody)

			mw := newGzipMiddleware(t, tt.configLevel, tt.configMinSize, tt.configContentTypes)

			req := httptest.NewRequest("GET", "/test-path", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			rec := httptest.NewRecorder()

			mw(handler).ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf(ExpectedStatusError, tt.expectedStatus, rec.Code)
			}

			expectedBody := tt.handlerBody
			if tt.expectCompression {
				assertCompressed(t, rec, expectedBody)
				return
			}
			assertUncompressed(t, rec, expectedBody)
		})
	}
}

func assertStatusOk(tb testing.TB, got int) {
	if got != http.StatusOK {
		tb.Fatalf(ExpectedStatusError, http.StatusOK, got)
	}
}

func assertContentEncoding(tb testing.TB, header string, expectGzip bool) {
	if expectGzip && header != "gzip" {
		tb.Fatalf("expected gzip header, got %q", header)
	}
	if !expectGzip && header != "" {
		tb.Fatalf("expected no gzip header, got %q", header)
	}
}

func BenchmarkGzipResponseTime(b *testing.B) {
	// Setup a large compressible body
	compressibleBody := []byte(largeBody)
	handlerType := ContentTypeJSON

	// Create a mock backend handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", handlerType)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(compressibleBody); err != nil {
			b.Fatalf(FailedToWriteError, err)
		}
	})

	mwCompressed := newGzipMiddleware(b, int(gzip.DefaultCompression), int(10), []string{ContentTypeJSON})

	mwUncompressed := newGzipMiddleware(b, int(gzip.DefaultCompression), int(len(compressibleBody)+1), []string{ContentTypeJSON})

	benchmarks := []struct {
		name         string
		middleware   Middleware
		expectedGzip bool
		acceptHeader string
	}{
		{"Compressed", mwCompressed, true, "gzip"},
		{"Uncompressed", mwUncompressed, false, ""},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest("GET", "/test-path", nil)
				req.Header.Set("Accept-Encoding", "gzip")
				rec := httptest.NewRecorder()
				bm.middleware(handler).ServeHTTP(rec, req)

				assertStatusOk(b, rec.Code)
				assertContentEncoding(b, rec.Header().Get("Content-Encoding"), bm.expectedGzip)
			}
		})
	}
}

func BenchmarkGzipCompressionRatio(b *testing.B) {
	tests := []struct {
		name        string
		body        []byte
		contentType string
	}{
		{
			name:        "Large JSON",
			body:        []byte(largeBody),
			contentType: ContentTypeJSON,
		},
		{
			name:        "Large HTML",
			body:        []byte(`<!DOCTYPE html><html><body><h1>Hello, World!</h1><p>This is a sample HTML page for testing compression ratios. It contains some repetitive text to ensure good compression. This is a sample HTML page for testing compression ratios. It contains some repetitive text to ensure good compression. This is a sample HTML page for testing compression ratios. It contains some repetitive text to ensure good compression.</p></body></html>`),
			contentType: "text/html",
		},
		{
			name:        "Small Text",
			body:        []byte(smallBody),
			contentType: "text/plain",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			handler := newMockHandler(b, tt.contentType, string(tt.body))

			mwCompressed := newGzipMiddleware(b, int(gzip.DefaultCompression), int(10), []string{tt.contentType})

			var originalSize int64
			var compressedSize int64

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest("GET", "/test-path", nil)
				req.Header.Set("Accept-Encoding", "gzip")
				rec := httptest.NewRecorder()
				mwCompressed(handler).ServeHTTP(rec, req)

				if rec.Code != http.StatusOK {
					b.Fatalf(ExpectedStatusError, http.StatusOK, rec.Code)
				}

				if rec.Header().Get("Content-Encoding") == "gzip" {
					originalSize = int64(len(tt.body))
					compressedSize = int64(len(rec.Body.Bytes()))
				} else {
					originalSize = int64(len(tt.body))
					compressedSize = int64(len(rec.Body.Bytes()))
				}
			}
			b.StopTimer()

			if originalSize > 0 {
				ratio := float64(compressedSize) / float64(originalSize)
				b.ReportMetric(ratio, "ratio")
				b.ReportMetric(float64(originalSize), "original_bytes")
				b.ReportMetric(float64(compressedSize), "compressed_bytes")
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
