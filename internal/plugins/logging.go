package plugins

import (
    "bufio"
    "log"
    "net"
    "net/http"
    "time"
)

// statusRecorder records HTTP status and bytes written
type statusRecorder struct {
    http.ResponseWriter
    status      int
    wroteHeader bool
}

func (sr *statusRecorder) WriteHeader(code int) {
    sr.status = code
    sr.wroteHeader = true
    sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
    if !sr.wroteHeader {
        sr.WriteHeader(http.StatusOK)
    }
    return sr.ResponseWriter.Write(b)
}

// Support http.Hijacker if underlying supports it (for websockets)
func (sr *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    if h, ok := sr.ResponseWriter.(http.Hijacker); ok {
        return h.Hijack()
    }
    return nil, nil, http.ErrNotSupported
}

// Support http.Flusher if underlying supports it
func (sr *statusRecorder) Flush() {
    if f, ok := sr.ResponseWriter.(http.Flusher); ok {
        f.Flush()
    }
}

func init() {
    RegisterBuiltin("logging", func(name string, cfg map[string]interface{}) (Middleware, error) {
        return func(next http.Handler) http.Handler {
            return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                start := time.Now()
                rec := &statusRecorder{ResponseWriter: w}
                next.ServeHTTP(rec, r)
                dur := time.Since(start)
                // Default to 200 if WriteHeader was never called
                status := rec.status
                if !rec.wroteHeader {
                    status = http.StatusOK
                }
                log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, status, dur)
            })
        }, nil
    })
}
