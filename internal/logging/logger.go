package logging

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/0xReLogic/Helios/internal/config"
)

type contextKey string

const (
	loggerKey    contextKey = "helios_logger"
	requestIDKey contextKey = "helios_request_id"
	traceIDKey   contextKey = "helios_trace_id"

	defaultRequestHeader = "X-Request-ID"
	defaultTraceHeader   = "X-Trace-ID"
)

type logFormat int

const (
	formatText logFormat = iota
	formatJSON
)

var (
	baseLogger   zerolog.Logger
	baseLoggerMu sync.RWMutex
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldUnit = time.Millisecond
	setBaseLogger(newLogger(os.Stdout, zerolog.InfoLevel, formatText, false))
}

// Init configures the global logger based on configuration values.
func Init(cfg config.LoggingConfig) {
	level := parseLevel(cfg.Level)
	format := parseFormat(cfg.Format)
	setBaseLogger(newLogger(os.Stdout, level, format, cfg.IncludeCaller))
}

func parseLevel(value string) zerolog.Level {
	switch strings.ToLower(value) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

func parseFormat(value string) logFormat {
	switch strings.ToLower(value) {
	case "json":
		return formatJSON
	default:
		return formatText
	}
}

func newLogger(writer io.Writer, level zerolog.Level, format logFormat, includeCaller bool) zerolog.Logger {
	var output io.Writer = writer
	if format == formatText {
		cw := zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339Nano,
			NoColor:    true,
		}
		output = &cw
	}

	builder := zerolog.New(output).Level(level).With().Timestamp()
	if includeCaller {
		builder = builder.CallerWithSkipFrameCount(1)
	}
	return builder.Logger()
}

func setBaseLogger(logger zerolog.Logger) {
	baseLoggerMu.Lock()
	baseLogger = logger
	baseLoggerMu.Unlock()
}

// L returns the base logger.
func L() zerolog.Logger {
	baseLoggerMu.RLock()
	logger := baseLogger
	baseLoggerMu.RUnlock()
	return logger
}

// WithContext returns a logger enriched with request scoped metadata.
func WithContext(ctx context.Context) zerolog.Logger {
	if ctx == nil {
		return L()
	}

	if logger, ok := ctx.Value(loggerKey).(zerolog.Logger); ok {
		return logger
	}

	reqID := RequestIDFromContext(ctx)
	traceID := TraceIDFromContext(ctx)
	if reqID == "" && traceID == "" {
		return L()
	}

	builder := L().With()
	if reqID != "" {
		builder = builder.Str("request_id", reqID)
	}
	if traceID != "" {
		builder = builder.Str("trace_id", traceID)
	}
	logger := builder.Logger()
	return logger
}

// RequestIDFromContext extracts the request identifier from context if present.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		return reqID
	}
	return ""
}

// TraceIDFromContext extracts the trace identifier from context if present.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// RequestHeaderName returns the configured request header name, falling back to default.
func RequestHeaderName(cfg config.LoggingConfig) string {
	header := strings.TrimSpace(cfg.RequestID.Header)
	if header == "" {
		return defaultRequestHeader
	}
	return header
}

// TraceHeaderName returns the configured trace header name, falling back to default.
func TraceHeaderName(cfg config.LoggingConfig) string {
	header := strings.TrimSpace(cfg.Trace.Header)
	if header == "" {
		return defaultTraceHeader
	}
	return header
}

func contextWithLogger(ctx context.Context, logger zerolog.Logger, reqID, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger != (zerolog.Logger{}) {
		ctx = context.WithValue(ctx, loggerKey, logger)
	}
	if reqID != "" {
		ctx = context.WithValue(ctx, requestIDKey, reqID)
	}
	if traceID != "" {
		ctx = context.WithValue(ctx, traceIDKey, traceID)
	}
	return ctx
}
