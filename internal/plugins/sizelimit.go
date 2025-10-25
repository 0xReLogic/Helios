package plugins

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"

	logging "github.com/0xReLogic/Helios/internal/logging"
)

const (
	// DefaultMaxRequestBody is the default maximum size for request bodies (10MB)
	DefaultMaxRequestBody = 10 * 1024 * 1024 // 10MB

	// DefaultMaxResponseBody is the default maximum size for response bodies (50MB)
	DefaultMaxResponseBody = 50 * 1024 * 1024 // 50MB
)

// limitedResponseWriter wraps http.ResponseWriter to track and limit bytes written
type limitedResponseWriter struct {
	http.ResponseWriter
	written      int64
	limit        int64
	limitReached bool
	wroteHeader  bool
	statusCode   int
	ctx          context.Context
}

// Write implements io.Writer, tracking bytes written and enforcing the limit
func (lrw *limitedResponseWriter) Write(b []byte) (int, error) {
	if lrw.limitReached {
		return 0, fmt.Errorf("response body exceeds limit of %d bytes", lrw.limit)
	}

	if err := lrw.checkLimit(b); err != nil {
		return 0, err
	}

	lrw.ensureHeaderWritten()

	n, err := lrw.ResponseWriter.Write(b)
	lrw.written += int64(n)
	return n, err
}

// checkLimit validates that writing b would not exceed the response body limit
func (lrw *limitedResponseWriter) checkLimit(b []byte) error {
	if lrw.written+int64(len(b)) <= lrw.limit {
		return nil
	}

	lrw.limitReached = true

	// Log the violation
	logging.WithContext(lrw.ctx).Warn().
		Int64("limit", lrw.limit).
		Int64("attempted", lrw.written+int64(len(b))).
		Int64("current", lrw.written).
		Str("type", "response").
		Msg("response body size limit exceeded")

	// If headers haven't been written yet, set the 413 status
	if !lrw.wroteHeader {
		lrw.statusCode = http.StatusRequestEntityTooLarge
		lrw.ResponseWriter.WriteHeader(http.StatusRequestEntityTooLarge)
		lrw.wroteHeader = true
	}

	return fmt.Errorf("response body exceeds limit of %d bytes", lrw.limit)
}

// ensureHeaderWritten writes the response header if it hasn't been written yet
func (lrw *limitedResponseWriter) ensureHeaderWritten() {
	if lrw.wroteHeader {
		return
	}

	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
	}

	lrw.ResponseWriter.WriteHeader(lrw.statusCode)
	lrw.wroteHeader = true
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

// parseByteLimit extracts and validates a byte limit from the configuration
func parseByteLimit(cfg map[string]interface{}, key string, defaultValue int64) (int64, error) {
	val, ok := cfg[key]
	if !ok {
		return defaultValue, nil
	}

	var limit int64
	switch v := val.(type) {
	case int:
		limit = int64(v)
	case int64:
		limit = v
	case float64:
		limit = int64(v)
	default:
		return 0, fmt.Errorf("%s must be a number, got %T", key, val)
	}

	if limit <= 0 {
		return 0, fmt.Errorf("%s must be positive, got %d", key, limit)
	}

	return limit, nil
}

// newSizeLimitMiddleware creates a new size limit middleware with the given configuration
func newSizeLimitMiddleware(name string, cfg map[string]interface{}) (Middleware, error) {
	// Parse and validate configuration
	maxRequestBody, err := parseByteLimit(cfg, "max_request_body", DefaultMaxRequestBody)
	if err != nil {
		return nil, err
	}

	maxResponseBody, err := parseByteLimit(cfg, "max_response_body", DefaultMaxResponseBody)
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check Content-Length header first for quick rejection
			if r.ContentLength > maxRequestBody {
				logging.WithContext(r.Context()).Warn().
					Int64("content_length", r.ContentLength).
					Int64("limit", maxRequestBody).
					Str("type", "request").
					Msg("request body size limit exceeded")
				
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}

			// Limit request body size for cases where Content-Length is not set
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
				ctx:            r.Context(),
			}

			// Call next handler with the limited response writer
			next.ServeHTTP(lrw, r)
		})
	}, nil
}

func init() {
	RegisterBuiltin("size_limit", newSizeLimitMiddleware)
}
