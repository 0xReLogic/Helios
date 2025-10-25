package plugins

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

// limitedResponseWriter wraps http.ResponseWriter to track and limit bytes written
type limitedResponseWriter struct {
	http.ResponseWriter
	written      int64
	limit        int64
	limitReached bool
	wroteHeader  bool
	statusCode   int
}

// Write implements io.Writer, tracking bytes written and enforcing the limit
func (lrw *limitedResponseWriter) Write(b []byte) (int, error) {
	if lrw.limitReached {
		// Already hit the limit, don't write more
		return 0, fmt.Errorf("response body exceeds limit of %d bytes", lrw.limit)
	}

	// Check if writing this chunk would exceed the limit
	if lrw.written+int64(len(b)) > lrw.limit {
		lrw.limitReached = true
		// If headers haven't been written yet, set the 413 status
		if !lrw.wroteHeader {
			lrw.statusCode = http.StatusRequestEntityTooLarge
			lrw.ResponseWriter.WriteHeader(http.StatusRequestEntityTooLarge)
			lrw.wroteHeader = true
		}
		return 0, fmt.Errorf("response body exceeds limit of %d bytes", lrw.limit)
	}

	// Write the header if not already written
	if !lrw.wroteHeader {
		if lrw.statusCode == 0 {
			lrw.statusCode = http.StatusOK
		}
		lrw.ResponseWriter.WriteHeader(lrw.statusCode)
		lrw.wroteHeader = true
	}

	n, err := lrw.ResponseWriter.Write(b)
	lrw.written += int64(n)
	return n, err
}

// WriteHeader records the status code but doesn't write it yet
// This allows us to override it with 413 if the body exceeds the limit
func (lrw *limitedResponseWriter) WriteHeader(statusCode int) {
	if lrw.wroteHeader {
		return
	}
	// Just record the status code, don't write it yet
	lrw.statusCode = statusCode
}

// Support http.Hijacker if underlying supports it (for websockets)
func (lrw *limitedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Support http.Flusher if underlying supports it
func (lrw *limitedResponseWriter) Flush() {
	if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func init() {
	RegisterBuiltin("size_limit", func(name string, cfg map[string]interface{}) (Middleware, error) {
		// Parse configuration with defaults
		// Default: 10MB for requests
		maxRequestBody := int64(10485760) // 10MB default
		if val, ok := cfg["max_request_body"]; ok {
			switch v := val.(type) {
			case int:
				maxRequestBody = int64(v)
			case int64:
				maxRequestBody = v
			case float64:
				maxRequestBody = int64(v)
			default:
				return nil, fmt.Errorf("max_request_body must be a number, got %T", val)
			}
		}

		// Default: 50MB for responses
		maxResponseBody := int64(52428800) // 50MB default
		if val, ok := cfg["max_response_body"]; ok {
			switch v := val.(type) {
			case int:
				maxResponseBody = int64(v)
			case int64:
				maxResponseBody = v
			case float64:
				maxResponseBody = int64(v)
			default:
				return nil, fmt.Errorf("max_response_body must be a number, got %T", val)
			}
		}

		// Validate limits
		if maxRequestBody <= 0 {
			return nil, fmt.Errorf("max_request_body must be positive, got %d", maxRequestBody)
		}
		if maxResponseBody <= 0 {
			return nil, fmt.Errorf("max_response_body must be positive, got %d", maxResponseBody)
		}

		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Limit request body size
				// http.MaxBytesReader returns a ReadCloser that stops reading once
				// the limit is exceeded and returns an error
				r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

				// Wrap response writer to limit response size
				lrw := &limitedResponseWriter{
					ResponseWriter: w,
					written:        0,
					limit:          maxResponseBody,
					limitReached:   false,
					wroteHeader:    false,
					statusCode:     0,
				}

				// Call next handler with the limited response writer
				next.ServeHTTP(lrw, r)

				// If we hit the response limit, log it or handle it
				// The limitedResponseWriter already sent 413 if limit was exceeded
			})
		}, nil
	})
}
