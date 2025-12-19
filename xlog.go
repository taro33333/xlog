// Package xlog provides a structured logging library built on top of log/slog.
// It features context propagation, environment-aware output, and zero external dependencies.
package xlog

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"
)

// Environment represents the runtime environment for logging configuration.
type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
	handler slog.Handler
}

// config holds the logger configuration.
type config struct {
	env         Environment
	level       slog.Level
	output      io.Writer
	addSource   bool
	timeFormat  string
	contextKeys []ContextKey
}

// Option is a functional option for configuring the logger.
type Option func(*config)

var (
	defaultLogger *Logger
	defaultMu     sync.RWMutex
)

func init() {
	// Initialize with a basic logger; users should call Init() to configure properly.
	defaultLogger = &Logger{
		Logger: slog.Default(),
	}
}

// WithEnvironment sets the logging environment.
func WithEnvironment(env Environment) Option {
	return func(c *config) {
		c.env = env
	}
}

// WithLevel sets the minimum logging level.
func WithLevel(level slog.Level) Option {
	return func(c *config) {
		c.level = level
	}
}

// WithOutput sets the output writer.
func WithOutput(w io.Writer) Option {
	return func(c *config) {
		c.output = w
	}
}

// WithSource enables or disables source code location in logs.
func WithSource(enabled bool) Option {
	return func(c *config) {
		c.addSource = enabled
	}
}

// WithTimeFormat sets the time format for development environment.
func WithTimeFormat(format string) Option {
	return func(c *config) {
		c.timeFormat = format
	}
}

// WithContextKeys sets the context keys to extract from context.
func WithContextKeys(keys ...ContextKey) Option {
	return func(c *config) {
		c.contextKeys = append(c.contextKeys, keys...)
	}
}

// Init initializes the global logger with the given options.
// It also updates slog.SetDefault and redirects standard log output.
func Init(opts ...Option) *Logger {
	cfg := &config{
		env:        Development,
		level:      slog.LevelInfo,
		output:     os.Stdout,
		addSource:  true,
		timeFormat: time.RFC3339,
		contextKeys: []ContextKey{
			TraceIDKey,
			UserIDKey,
			RequestIDKey,
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	var baseHandler slog.Handler
	handlerOpts := &slog.HandlerOptions{
		AddSource: cfg.addSource,
		Level:     cfg.level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize time format for development
			if a.Key == slog.TimeKey && cfg.env == Development {
				if t, ok := a.Value.Any().(time.Time); ok {
					return slog.String(slog.TimeKey, t.Format(cfg.timeFormat))
				}
			}
			return a
		},
	}

	switch cfg.env {
	case Production:
		baseHandler = slog.NewJSONHandler(cfg.output, handlerOpts)
	default:
		baseHandler = NewColorHandler(cfg.output, handlerOpts)
	}

	// Wrap with context handler
	ctxHandler := NewContextHandler(baseHandler, cfg.contextKeys...)

	logger := &Logger{
		Logger:  slog.New(ctxHandler),
		handler: ctxHandler,
	}

	// Set as default
	defaultMu.Lock()
	defaultLogger = logger
	defaultMu.Unlock()

	// Update slog default
	slog.SetDefault(logger.Logger)

	// Redirect standard log output to slog
	log.SetOutput(&slogWriter{logger: logger.Logger})
	log.SetFlags(0)

	return logger
}

// Default returns the default logger.
func Default() *Logger {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultLogger
}

// slogWriter adapts slog.Logger to io.Writer for standard log integration.
type slogWriter struct {
	logger *slog.Logger
}

func (w *slogWriter) Write(p []byte) (n int, err error) {
	// Remove trailing newline if present
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	w.logger.Info(msg)
	return len(p), nil
}

// callerSkip is the number of stack frames to skip when determining the caller.
// This is carefully calibrated to account for the wrapper functions.
const callerSkip = 3

// logWithCaller logs a message with correct caller information.
func logWithCaller(ctx context.Context, logger *slog.Logger, level slog.Level, msg string, args ...any) {
	if !logger.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(callerSkip, pcs[:])

	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)

	_ = logger.Handler().Handle(ctx, r)
}

// Debug logs at DEBUG level with context.
func Debug(ctx context.Context, msg string, args ...any) {
	logWithCaller(ctx, Default().Logger, slog.LevelDebug, msg, args...)
}

// Info logs at INFO level with context.
func Info(ctx context.Context, msg string, args ...any) {
	logWithCaller(ctx, Default().Logger, slog.LevelInfo, msg, args...)
}

// Warn logs at WARN level with context.
func Warn(ctx context.Context, msg string, args ...any) {
	logWithCaller(ctx, Default().Logger, slog.LevelWarn, msg, args...)
}

// Error logs at ERROR level with context.
func Error(ctx context.Context, msg string, args ...any) {
	logWithCaller(ctx, Default().Logger, slog.LevelError, msg, args...)
}

// With returns a new Logger with the given attributes.
func With(args ...any) *Logger {
	l := Default()
	return &Logger{
		Logger:  l.Logger.With(args...),
		handler: l.handler,
	}
}

// WithGroup returns a new Logger with the given group name.
func WithGroup(name string) *Logger {
	l := Default()
	return &Logger{
		Logger:  l.Logger.WithGroup(name),
		handler: l.handler,
	}
}

// Logger methods

// Debug logs at DEBUG level with context.
func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	logWithCaller(ctx, l.Logger, slog.LevelDebug, msg, args...)
}

// Info logs at INFO level with context.
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	logWithCaller(ctx, l.Logger, slog.LevelInfo, msg, args...)
}

// Warn logs at WARN level with context.
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	logWithCaller(ctx, l.Logger, slog.LevelWarn, msg, args...)
}

// Error logs at ERROR level with context.
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	logWithCaller(ctx, l.Logger, slog.LevelError, msg, args...)
}

// With returns a new Logger with the given attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger:  l.Logger.With(args...),
		handler: l.handler,
	}
}

// WithGroup returns a new Logger with the given group name.
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		Logger:  l.Logger.WithGroup(name),
		handler: l.handler,
	}
}
