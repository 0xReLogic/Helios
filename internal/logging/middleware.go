package logging

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/0xReLogic/Helios/internal/config"
)

// RequestContextMiddleware injects request/trace identifiers into the request context.
func RequestContextMiddleware(cfg config.LoggingConfig) func(http.Handler) http.Handler {
	requestHeader := RequestHeaderName(cfg)
	traceHeader := TraceHeaderName(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			var requestID string
			if cfg.RequestID.Enabled {
				requestID = strings.TrimSpace(r.Header.Get(requestHeader))
				if requestID == "" {
					requestID = generateIdentifier("req")
					r.Header.Set(requestHeader, requestID)
				}
				w.Header().Set(requestHeader, requestID)
			}

			var traceID string
			if cfg.Trace.Enabled {
				traceID = strings.TrimSpace(r.Header.Get(traceHeader))
				if traceID == "" {
					traceID = generateIdentifier("trace")
					r.Header.Set(traceHeader, traceID)
				}
				w.Header().Set(traceHeader, traceID)
			}

			logger := WithContext(ctx)
			if requestID != "" {
				logger = logger.With().Str("request_id", requestID).Logger()
			}
			if traceID != "" {
				logger = logger.With().Str("trace_id", traceID).Logger()
			}

			ctx = contextWithLogger(ctx, logger, requestID, traceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func generateIdentifier(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}
