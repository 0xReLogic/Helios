package zerolog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Field names mirror the real zerolog defaults for compatibility.
var (
	LevelFieldName   = "level"
	TimeFieldName    = "time"
	MessageFieldName = "message"
	CallerFieldName  = "caller"
	ErrorFieldName   = "error"

	TimeFieldFormat   = time.RFC3339Nano
	DurationFieldUnit = time.Millisecond
)

// Level defines log severity.
type Level int8

const (
	TraceLevel Level = -1
	DebugLevel Level = 0
	InfoLevel  Level = 1
	WarnLevel  Level = 2
	ErrorLevel Level = 3
	FatalLevel Level = 4
	PanicLevel Level = 5
)

type fieldMap map[string]interface{}

// Logger is a minimal structured logger compatible with zerolog semantics.
type Logger struct {
	writer           io.Writer
	level            Level
	context          *fieldMap
	includeTimestamp bool
	includeCaller    bool
	callerSkip       int
}

// New constructs a new logger writing to the provided writer.
func New(w io.Writer) Logger {
	if w == nil {
		w = os.Stdout
	}
	ctx := fieldMap{}
	return Logger{
		writer:  w,
		level:   InfoLevel,
		context: &ctx,
	}
}

// Level adjusts the minimum enabled logging level for the logger.
func (l Logger) Level(level Level) Logger {
	l.level = level
	return l
}

// With prepares a builder that can augment the logger's base context.
func (l Logger) With() Context {
	ctx := Context{
		logger:           l,
		fields:           l.copyContext(),
		includeTimestamp: l.includeTimestamp,
		includeCaller:    l.includeCaller,
		callerSkip:       l.callerSkip,
	}
	return ctx
}

func (l Logger) copyContext() fieldMap {
	result := fieldMap{}
	if l.context != nil {
		for k, v := range *l.context {
			result[k] = v
		}
	}
	return result
}

func (l Logger) newEvent(level Level, terminal bool) *Event {
	evt := &Event{
		logger:           l,
		level:            level,
		includeTimestamp: l.includeTimestamp,
		includeCaller:    l.includeCaller,
		callerSkip:       l.callerSkip,
		terminal:         terminal,
	}
	return evt
}

// Info starts an info-level log event.
func (l Logger) Info() *Event { return l.newEvent(InfoLevel, false) }

// Warn starts a warn-level log event.
func (l Logger) Warn() *Event { return l.newEvent(WarnLevel, false) }

// Error starts an error-level log event.
func (l Logger) Error() *Event { return l.newEvent(ErrorLevel, false) }

// Debug starts a debug-level log event.
func (l Logger) Debug() *Event { return l.newEvent(DebugLevel, false) }

// Trace starts a trace-level log event.
func (l Logger) Trace() *Event { return l.newEvent(TraceLevel, false) }

// Fatal starts a fatal-level log event which will exit after logging.
func (l Logger) Fatal() *Event { return l.newEvent(FatalLevel, true) }

// WithLevel starts an event at an arbitrary level.
func (l Logger) WithLevel(level Level) *Event { return l.newEvent(level, false) }

// Context represents a logger builder used to configure base fields.
type Context struct {
	logger           Logger
	fields           fieldMap
	includeTimestamp bool
	includeCaller    bool
	callerSkip       int
}

// Timestamp enables automatic timestamp inclusion for derived loggers.
func (c Context) Timestamp() Context {
	c.includeTimestamp = true
	return c
}

// CallerWithSkipFrameCount enables caller information with the provided skip depth.
func (c Context) CallerWithSkipFrameCount(skip int) Context {
	c.includeCaller = true
	c.callerSkip = skip
	return c
}

// Str attaches a string field to the context builder.
func (c Context) Str(key, value string) Context {
	c.fields[key] = value
	return c
}

// Logger finalises the builder and returns an updated logger.
func (c Context) Logger() Logger {
	copied := fieldMap{}
	for k, v := range c.fields {
		copied[k] = v
	}
	c.logger.context = &copied
	c.logger.includeTimestamp = c.includeTimestamp
	c.logger.includeCaller = c.includeCaller
	c.logger.callerSkip = c.callerSkip
	return c.logger
}

// Strs attaches a string slice field to the context builder.
func (c Context) Strs(key string, values []string) Context {
	copied := make([]string, len(values))
	copy(copied, values)
	c.fields[key] = copied
	return c
}

// Event represents a structured log entry under construction.
type Event struct {
	logger           Logger
	level            Level
	fields           fieldMap
	err              error
	includeTimestamp bool
	includeCaller    bool
	callerSkip       int
	terminal         bool
}

func (e *Event) ensureFields() {
	if e.fields == nil {
		e.fields = fieldMap{}
	}
}

// Str adds a string field to the event.
func (e *Event) Str(key, value string) *Event {
	if e == nil {
		return e
	}
	e.ensureFields()
	e.fields[key] = value
	return e
}

// Strs adds a string slice field to the event.
func (e *Event) Strs(key string, values []string) *Event {
	if e == nil {
		return e
	}
	e.ensureFields()
	copied := make([]string, len(values))
	copy(copied, values)
	e.fields[key] = copied
	return e
}

// Int adds an integer field to the event.
func (e *Event) Int(key string, value int) *Event {
	if e == nil {
		return e
	}
	e.ensureFields()
	e.fields[key] = value
	return e
}

// Uint32 adds an unsigned 32-bit integer field to the event.
func (e *Event) Uint32(key string, value uint32) *Event {
	if e == nil {
		return e
	}
	e.ensureFields()
	e.fields[key] = value
	return e
}

// Float64 adds a float field to the event.
func (e *Event) Float64(key string, value float64) *Event {
	if e == nil {
		return e
	}
	e.ensureFields()
	e.fields[key] = value
	return e
}

// Dur records a duration field scaled by DurationFieldUnit.
func (e *Event) Dur(key string, value time.Duration) *Event {
	if e == nil {
		return e
	}
	e.ensureFields()
	scalar := float64(value) / float64(DurationFieldUnit)
	e.fields[key] = scalar
	return e
}

// Bytes stores a byte slice field as a base64 string.
func (e *Event) Bytes(key string, value []byte) *Event {
	if e == nil {
		return e
	}
	e.ensureFields()
	e.fields[key] = fmt.Sprintf("%x", value)
	return e
}

// Err records an error with the event.
func (e *Event) Err(err error) *Event {
	if e == nil {
		return e
	}
	e.err = err
	return e
}

// Msg writes the event with the supplied message.
func (e *Event) Msg(message string) {
	if e == nil {
		return
	}
	if e.level < e.logger.level {
		if e.terminal {
			os.Exit(1)
		}
		return
	}

	payload := fieldMap{}
	if e.logger.context != nil {
		for k, v := range *e.logger.context {
			payload[k] = v
		}
	}
	for k, v := range e.fields {
		payload[k] = v
	}

	payload[LevelFieldName] = levelString(e.level)
	if e.includeTimestamp {
		payload[TimeFieldName] = time.Now().UTC().Format(TimeFieldFormat)
	}
	if e.includeCaller {
		if caller := callerLocation(e.callerSkip); caller != "" {
			payload[CallerFieldName] = caller
		}
	}
	if message != "" {
		payload[MessageFieldName] = message
	}
	if e.err != nil {
		payload[ErrorFieldName] = e.err.Error()
	}

	data, err := json.Marshal(payload)
	if err == nil {
		if !bytes.HasSuffix(data, []byte("\n")) {
			data = append(data, '\n')
		}
		_, _ = e.logger.writer.Write(data)
	}

	if e.terminal {
		os.Exit(1)
	}
}

func callerLocation(skip int) string {
	// additional frames: Event.Msg and the logging helper methods.
	const internalFrames = 2
	_, file, line, ok := runtime.Caller(skip + internalFrames)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s:%d", trimFile(file), line)
}

func trimFile(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 && idx+1 < len(path) {
		return path[idx+1:]
	}
	return path
}

func levelString(level Level) string {
	switch level {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	default:
		return "info"
	}
}

// ConsoleWriter converts JSON payloads into key=value text lines similar to zerolog's console writer.
type ConsoleWriter struct {
	Out        io.Writer
	TimeFormat string
	NoColor    bool
}

// Write implements io.Writer.
func (cw *ConsoleWriter) Write(p []byte) (int, error) {
	if cw == nil {
		return 0, fmt.Errorf("nil ConsoleWriter")
	}
	if cw.Out == nil {
		cw.Out = os.Stdout
	}
	line := bytes.TrimSpace(p)
	if len(line) == 0 {
		return cw.Out.Write(p)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(line, &payload); err != nil {
		return cw.Out.Write(append(line, '\n'))
	}

	order := []string{TimeFieldName, LevelFieldName}
	builder := strings.Builder{}

	for _, key := range order {
		if val, ok := payload[key]; ok {
			appendField(&builder, key, val)
			builder.WriteByte(' ')
			delete(payload, key)
		}
	}

	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == MessageFieldName {
			continue
		}
		appendField(&builder, key, payload[key])
		builder.WriteByte(' ')
	}

	output := strings.TrimSpace(builder.String())
	if msg, ok := payload[MessageFieldName]; ok {
		if output != "" {
			output += " "
		}
		output += fmt.Sprintf("%s=%s", MessageFieldName, formatValue(msg))
	}
	output = strings.TrimSpace(output)
	output += "\n"

	return cw.Out.Write([]byte(output))
}

func appendField(builder *strings.Builder, key string, value interface{}) {
	builder.WriteString(key)
	builder.WriteByte('=')
	builder.WriteString(formatValue(value))
}

func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		if strings.ContainsAny(v, " \t") {
			return strconv.Quote(v)
		}
		return v
	case fmt.Stringer:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case bool:
		return strconv.FormatBool(v)
	case []string:
		data, _ := json.Marshal(v)
		return string(data)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	}
}
