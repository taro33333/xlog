package xlog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
)

// ContextKey is a type for context keys used by xlog.
type ContextKey string

// Predefined context keys for common use cases.
const (
	TraceIDKey   ContextKey = "trace_id"
	UserIDKey    ContextKey = "user_id"
	RequestIDKey ContextKey = "request_id"
	SessionIDKey ContextKey = "session_id"
	SpanIDKey    ContextKey = "span_id"
)

// ContextHandler wraps a slog.Handler and extracts values from context.
type ContextHandler struct {
	handler slog.Handler
	keys    []ContextKey
}

// NewContextHandler creates a new ContextHandler that extracts the specified keys from context.
func NewContextHandler(handler slog.Handler, keys ...ContextKey) *ContextHandler {
	return &ContextHandler{
		handler: handler,
		keys:    keys,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle extracts context values and adds them to the record before delegating.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Extract values from context and add as attributes
	// Use a pre-allocated slice to minimize allocations
	attrs := make([]slog.Attr, 0, len(h.keys))

	for _, key := range h.keys {
		if v := ctx.Value(key); v != nil {
			attrs = append(attrs, slog.Any(string(key), v))
		}
	}

	if len(attrs) > 0 {
		// Clone the record and add context attributes at the beginning
		r2 := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		r2.AddAttrs(attrs...)
		r.Attrs(func(a slog.Attr) bool {
			r2.AddAttrs(a)
			return true
		})
		return h.handler.Handle(ctx, r2)
	}

	return h.handler.Handle(ctx, r)
}

// WithAttrs returns a new handler with the given attributes.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		handler: h.handler.WithAttrs(attrs),
		keys:    h.keys,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{
		handler: h.handler.WithGroup(name),
		keys:    h.keys,
	}
}

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[37m"
	colorBold   = "\033[1m"
)

// ColorHandler is a development-friendly handler with colored output.
type ColorHandler struct {
	opts      *slog.HandlerOptions
	output    io.Writer
	mu        *sync.Mutex
	attrs     []slog.Attr
	groups    []string
	preformat string
}

// NewColorHandler creates a new ColorHandler for development environments.
func NewColorHandler(output io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &ColorHandler{
		opts:   opts,
		output: output,
		mu:     &sync.Mutex{},
		attrs:  make([]slog.Attr, 0),
		groups: make([]string, 0),
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *ColorHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle formats and writes the log record with colors.
func (h *ColorHandler) Handle(_ context.Context, r slog.Record) error {
	// Get level color
	levelColor := h.levelColor(r.Level)
	levelStr := h.levelString(r.Level)

	// Build the log line using a byte slice for efficiency
	buf := make([]byte, 0, 256)

	// Timestamp
	if !r.Time.IsZero() {
		buf = append(buf, colorGray...)
		if h.opts.ReplaceAttr != nil {
			a := h.opts.ReplaceAttr(nil, slog.Time(slog.TimeKey, r.Time))
			buf = append(buf, a.Value.String()...)
		} else {
			buf = append(buf, r.Time.Format("2006-01-02 15:04:05")...)
		}
		buf = append(buf, colorReset...)
		buf = append(buf, ' ')
	}

	// Level
	buf = append(buf, levelColor...)
	buf = append(buf, levelStr...)
	buf = append(buf, colorReset...)
	buf = append(buf, ' ')

	// Source
	if h.opts.AddSource && r.PC != 0 {
		buf = append(buf, colorCyan...)
		buf = append(buf, h.formatSource(r.PC)...)
		buf = append(buf, colorReset...)
		buf = append(buf, ' ')
	}

	// Message
	buf = append(buf, colorBold...)
	buf = append(buf, r.Message...)
	buf = append(buf, colorReset...)

	// Pre-formatted attrs from WithAttrs
	if h.preformat != "" {
		buf = append(buf, ' ')
		buf = append(buf, h.preformat...)
	}

	// Record attrs
	r.Attrs(func(a slog.Attr) bool {
		buf = append(buf, ' ')
		buf = h.appendAttr(buf, a, h.groups)
		return true
	})

	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.output.Write(buf)
	return err
}

// WithAttrs returns a new handler with the given attributes.
func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)

	// Pre-format the attributes
	var buf []byte
	for _, a := range attrs {
		if len(buf) > 0 {
			buf = append(buf, ' ')
		}
		buf = h.appendAttr(buf, a, h.groups)
	}

	preformat := h.preformat
	if len(buf) > 0 {
		if preformat != "" {
			preformat += " "
		}
		preformat += string(buf)
	}

	return &ColorHandler{
		opts:      h.opts,
		output:    h.output,
		mu:        h.mu,
		attrs:     newAttrs,
		groups:    h.groups,
		preformat: preformat,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *ColorHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)

	return &ColorHandler{
		opts:      h.opts,
		output:    h.output,
		mu:        h.mu,
		attrs:     h.attrs,
		groups:    newGroups,
		preformat: h.preformat,
	}
}

func (h *ColorHandler) levelColor(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return colorRed
	case level >= slog.LevelWarn:
		return colorYellow
	case level >= slog.LevelInfo:
		return colorGreen
	default:
		return colorBlue
	}
}

func (h *ColorHandler) levelString(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "ERR"
	case level >= slog.LevelWarn:
		return "WRN"
	case level >= slog.LevelInfo:
		return "INF"
	default:
		return "DBG"
	}
}

func (h *ColorHandler) formatSource(pc uintptr) string {
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	if frame.File != "" {
		// Extract just the filename, not the full path
		short := frame.File
		for i := len(frame.File) - 1; i > 0; i-- {
			if frame.File[i] == '/' {
				short = frame.File[i+1:]
				break
			}
		}
		return fmt.Sprintf("%s:%d", short, frame.Line)
	}
	return ""
}

func (h *ColorHandler) appendAttr(buf []byte, a slog.Attr, groups []string) []byte {
	// Skip empty attrs
	if a.Equal(slog.Attr{}) {
		return buf
	}

	// Handle ReplaceAttr if set
	if h.opts.ReplaceAttr != nil {
		a = h.opts.ReplaceAttr(groups, a)
		if a.Equal(slog.Attr{}) {
			return buf
		}
	}

	// Build key with groups
	key := a.Key
	if len(groups) > 0 {
		for _, g := range groups {
			key = g + "." + key
		}
	}

	// Handle group values
	if a.Value.Kind() == slog.KindGroup {
		groupAttrs := a.Value.Group()
		if len(groupAttrs) == 0 {
			return buf
		}
		newGroups := append(groups, a.Key)
		for i, ga := range groupAttrs {
			if i > 0 || len(buf) > 0 {
				buf = append(buf, ' ')
			}
			buf = h.appendAttr(buf, ga, newGroups)
		}
		return buf
	}

	// Format key=value
	buf = append(buf, colorPurple...)
	buf = append(buf, key...)
	buf = append(buf, colorReset...)
	buf = append(buf, '=')
	buf = append(buf, formatValue(a.Value)...)

	return buf
}

func formatValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		// Quote strings with spaces
		if needsQuoting(s) {
			return fmt.Sprintf("%q", s)
		}
		return s
	case slog.KindTime:
		return v.Time().Format("2006-01-02T15:04:05.000Z07:00")
	case slog.KindDuration:
		return v.Duration().String()
	default:
		return fmt.Sprintf("%v", v.Any())
	}
}

func needsQuoting(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '"' || r == '=' || r == '\n' || r == '\r' || r == '\t' {
			return true
		}
	}
	return false
}

// WithContext adds the specified key-value pair to the context.
func WithContext(ctx context.Context, key ContextKey, value any) context.Context {
	return context.WithValue(ctx, key, value)
}

// WithTraceID adds a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
