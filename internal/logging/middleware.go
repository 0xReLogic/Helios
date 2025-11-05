package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/rs/zerolog"
)

// RequestContextMiddleware injects request/trace identifiers into the request context.
func RequestContextMiddleware(cfg config.LoggingConfig) func(http.Handler) http.Handler {
	requestHeader := RequestHeaderName(cfg)
	traceHeader := TraceHeaderName(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			requestID := handleRequestID(r, w, cfg, requestHeader)
			traceID := handleTraceID(r, w, cfg, traceHeader)
			logger := enrichLogger(ctx, requestID, traceID)

			ctx = contextWithLogger(ctx, logger, requestID, traceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func handleRequestID(r *http.Request, w http.ResponseWriter, cfg config.LoggingConfig, header string) string {
	if !cfg.RequestID.Enabled {
		return ""
	}

	requestID := strings.TrimSpace(r.Header.Get(header))
	if requestID == "" {
		requestID = generateIdentifier("req")
		r.Header.Set(header, requestID)
	}
	w.Header().Set(header, requestID)
	return requestID
}

func handleTraceID(r *http.Request, w http.ResponseWriter, cfg config.LoggingConfig, header string) string {
	if !cfg.Trace.Enabled {
		return ""
	}

	traceID := strings.TrimSpace(r.Header.Get(header))
	if traceID == "" {
		traceID = generateIdentifier("trace")
		r.Header.Set(header, traceID)
	}
	w.Header().Set(header, traceID)
	return traceID
}

func enrichLogger(ctx context.Context, requestID, traceID string) *zerolog.Logger {
	logger := WithContext(ctx)
	if requestID != "" {
		enriched := logger.With().Str("request_id", requestID).Logger()
		logger = &enriched
	}
	if traceID != "" {
		enriched := logger.With().Str("trace_id", traceID).Logger()
		logger = &enriched
	}
	return logger
}

func generateIdentifier(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		// Log the error before falling back to timestamp-based ID
		L().Warn().Err(err).Msg("failed to generate random identifier, falling back to timestamp")
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}
