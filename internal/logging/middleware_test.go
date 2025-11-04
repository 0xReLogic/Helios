package logging

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"github.com/0xReLogic/Helios/internal/config"
)

const (
	testCustomReqHeader   = "X-Custom-Req"
	testCustomTraceHeader = "X-Custom-Trace"
	testReqID             = "req-1"
	testTraceID           = "trace-1"
)

func swapLoggerForTest(logger zerolog.Logger) func() {
	baseLoggerMu.Lock()
	previous := baseLogger
	copyLogger := logger
	baseLogger = &copyLogger
	baseLoggerMu.Unlock()
	return func() {
		baseLoggerMu.Lock()
		baseLogger = previous
		baseLoggerMu.Unlock()
	}
}

func firstLine(b []byte) []byte {
	if idx := bytes.IndexByte(b, '\n'); idx >= 0 {
		return b[:idx]
	}
	return b
}

func TestRequestContextMiddleware_GeneratesIdentifiers(t *testing.T) {
	cfg := config.LoggingConfig{
		RequestID: config.RequestIDConfig{Enabled: true},
		Trace:     config.TraceConfig{Enabled: true},
	}

	buffer := bytes.Buffer{}
	restore := swapLoggerForTest(newLogger(&buffer, zerolog.InfoLevel, formatJSON, false))
	defer restore()

	mw := RequestContextMiddleware(cfg)

	var capturedRequestID, capturedTraceID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = RequestIDFromContext(r.Context())
		capturedTraceID = TraceIDFromContext(r.Context())
		WithContext(r.Context()).Info().Msg("test")
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	mw(handler).ServeHTTP(rr, req)

	if capturedRequestID == "" {
		t.Fatal("expected generated request id")
	}
	if capturedTraceID == "" {
		t.Fatal("expected generated trace id")
	}

	if got := rr.Header().Get(defaultRequestHeader); got != capturedRequestID {
		t.Fatalf("expected response header request id %q, got %q", capturedRequestID, got)
	}
	if got := rr.Header().Get(defaultTraceHeader); got != capturedTraceID {
		t.Fatalf("expected response header trace id %q, got %q", capturedTraceID, got)
	}

	line := firstLine(buffer.Bytes())
	if len(line) == 0 {
		t.Fatal("expected log output")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(line, &payload); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if payload["request_id"] != capturedRequestID {
		t.Fatalf("expected log request_id %q, got %v", capturedRequestID, payload["request_id"])
	}
	if payload["trace_id"] != capturedTraceID {
		t.Fatalf("expected log trace_id %q, got %v", capturedTraceID, payload["trace_id"])
	}
}

func TestRequestContextMiddleware_RespectsHeaders(t *testing.T) {
	cfg := config.LoggingConfig{
		RequestID: config.RequestIDConfig{Enabled: true, Header: testCustomReqHeader},
		Trace:     config.TraceConfig{Enabled: true, Header: testCustomTraceHeader},
	}

	mw := RequestContextMiddleware(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got != testReqID {
			t.Fatalf("expected request id %s, got %s", testReqID, got)
		}
		if got := TraceIDFromContext(r.Context()); got != testTraceID {
			t.Fatalf("expected trace id %s, got %s", testTraceID, got)
		}
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(testCustomReqHeader, testReqID)
	req.Header.Set(testCustomTraceHeader, testTraceID)

	mw(handler).ServeHTTP(rr, req)

	if got := rr.Header().Get(testCustomReqHeader); got != testReqID {
		t.Fatalf("expected response header %s, got %s", testReqID, got)
	}
	if got := rr.Header().Get(testCustomTraceHeader); got != testTraceID {
		t.Fatalf("expected response header %s, got %s", testTraceID, got)
	}
}
