# xlog

A high-performance, zero-dependency logging library for Go built on `log/slog`.

## Features

- **Zero External Dependencies**: Built entirely on Go's standard library (`log/slog`)
- **Context Propagation**: Automatically extracts TraceID, UserID, RequestID from context
- **Environment-Aware**: One-line switch between development (colored text) and production (JSON)
- **Functional Options Pattern**: Extensible configuration
- **Correct Caller Information**: Accurate file/line reporting without stack frame issues
- **Thread-Safe**: Safe for concurrent use
- **High Performance**: Minimal memory allocations

## Installation

```bash
go get github.com/taro33333/xlog
```

## Quick Start

```go
package main

import (
    "context"
    "log/slog"

    "github.com/taro33333/xlog"
)

func main() {
    // Initialize logger (development mode by default)
    xlog.Init()

    ctx := context.Background()
    xlog.Info(ctx, "server started", "port", 8080)
}
```

## Configuration

### Environment-Based Configuration

```go
// Development: Colored text output
xlog.Init(
    xlog.WithEnvironment(xlog.Development),
    xlog.WithLevel(slog.LevelDebug),
)

// Production: JSON structured logs
xlog.Init(
    xlog.WithEnvironment(xlog.Production),
    xlog.WithLevel(slog.LevelInfo),
)
```

### Available Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithEnvironment(env)` | Set logging environment | `Development` |
| `WithLevel(level)` | Set minimum log level | `slog.LevelInfo` |
| `WithOutput(w)` | Set output writer | `os.Stdout` |
| `WithSource(bool)` | Enable/disable source location | `true` |
| `WithTimeFormat(fmt)` | Set time format (dev mode) | `time.RFC3339` |
| `WithContextKeys(keys...)` | Set context keys to extract | TraceID, UserID, RequestID |

## Context Propagation

xlog automatically extracts values from context and adds them to log output.

```go
// Add trace information to context
ctx := context.Background()
ctx = xlog.WithTraceID(ctx, "abc-123-def")
ctx = xlog.WithUserID(ctx, "user-456")
ctx = xlog.WithRequestID(ctx, "req-789")

// Logs will automatically include trace_id, user_id, request_id
xlog.Info(ctx, "processing request", "action", "create")
```

### Custom Context Keys

```go
// Define custom context keys
const MyCustomKey xlog.ContextKey = "custom_field"

// Initialize with custom keys
xlog.Init(
    xlog.WithContextKeys(
        xlog.TraceIDKey,
        xlog.UserIDKey,
        MyCustomKey,
    ),
)

// Add to context
ctx = xlog.WithContext(ctx, MyCustomKey, "custom_value")
```

### Predefined Context Keys

| Key | Description |
|-----|-------------|
| `xlog.TraceIDKey` | Distributed tracing ID |
| `xlog.UserIDKey` | User identifier |
| `xlog.RequestIDKey` | Request identifier |
| `xlog.SessionIDKey` | Session identifier |
| `xlog.SpanIDKey` | Span identifier |

## Logging API

All logging functions take `context.Context` as the first argument:

```go
xlog.Debug(ctx, "debug message", "key", "value")
xlog.Info(ctx, "info message", "key", "value")
xlog.Warn(ctx, "warning message", "key", "value")
xlog.Error(ctx, "error message", "err", err)
```

### Logger Instance

```go
// Get the default logger
logger := xlog.Default()

// Create a logger with additional attributes
logger = xlog.With("service", "api", "version", "1.0")

// Create a logger with a group
logger = xlog.WithGroup("http")
logger.Info(ctx, "request received", "method", "GET")
```

## Output Examples

### Development Mode

```
2024-01-15 10:30:45 INF main.go:25 server started port=8080
2024-01-15 10:30:46 INF handler.go:42 processing request trace_id=abc-123 user_id=user-456 action=create
2024-01-15 10:30:47 ERR handler.go:55 failed to process err="connection refused"
```

### Production Mode (JSON)

```json
{"time":"2024-01-15T10:30:45Z","level":"INFO","source":{"file":"main.go","line":25},"msg":"server started","port":8080}
{"time":"2024-01-15T10:30:46Z","level":"INFO","source":{"file":"handler.go","line":42},"msg":"processing request","trace_id":"abc-123","user_id":"user-456","action":"create"}
```

## Standard Library Integration

xlog redirects output from the standard `log` package:

```go
import "log"

xlog.Init()

// This will be captured by xlog
log.Println("message from standard log")
```

## HTTP Middleware Example

```go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract or generate request ID
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        // Add to context
        ctx := r.Context()
        ctx = xlog.WithRequestID(ctx, requestID)
        ctx = xlog.WithTraceID(ctx, r.Header.Get("X-Trace-ID"))

        xlog.Info(ctx, "request started",
            "method", r.Method,
            "path", r.URL.Path,
        )

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Performance

xlog is designed for high-performance scenarios:

- Pre-allocated buffers for log formatting
- Minimal interface{} boxing
- Efficient context value extraction
- sync.Mutex only for write operations

## Thread Safety

xlog is fully thread-safe. All exported functions and methods can be safely called from multiple goroutines.

## License

MIT License
