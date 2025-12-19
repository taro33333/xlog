package xlog_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/taro33333/xlog"
)

func TestInit(t *testing.T) {
	var buf bytes.Buffer
	_ = xlog.Init(
		xlog.WithEnvironment(xlog.Development),
		xlog.WithLevel(slog.LevelDebug),
		xlog.WithOutput(&buf),
	)

	ctx := context.Background()
	xlog.Info(ctx, "test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("expected output to contain key and value, got: %s", output)
	}
}

func TestContextPropagation(t *testing.T) {
	var buf bytes.Buffer
	_ = xlog.Init(
		xlog.WithEnvironment(xlog.Production),
		xlog.WithOutput(&buf),
		xlog.WithSource(false),
	)

	ctx := context.Background()
	ctx = xlog.WithTraceID(ctx, "trace-123")
	ctx = xlog.WithUserID(ctx, "user-456")

	xlog.Info(ctx, "test with context")

	output := buf.String()
	if !strings.Contains(output, "trace-123") {
		t.Errorf("expected output to contain trace_id, got: %s", output)
	}
	if !strings.Contains(output, "user-456") {
		t.Errorf("expected output to contain user_id, got: %s", output)
	}
}

func TestProductionJSON(t *testing.T) {
	var buf bytes.Buffer
	_ = xlog.Init(
		xlog.WithEnvironment(xlog.Production),
		xlog.WithOutput(&buf),
		xlog.WithSource(false),
	)

	ctx := context.Background()
	xlog.Info(ctx, "json test", "count", 42)

	output := buf.String()
	// Should be valid JSON format
	if !strings.HasPrefix(output, "{") {
		t.Errorf("expected JSON output, got: %s", output)
	}
	if !strings.Contains(output, `"count":42`) {
		t.Errorf("expected output to contain count:42, got: %s", output)
	}
}

func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	_ = xlog.Init(
		xlog.WithEnvironment(xlog.Development),
		xlog.WithOutput(&buf),
		xlog.WithSource(false),
	)

	logger := xlog.With("service", "api")
	ctx := context.Background()
	logger.Info(ctx, "with attrs")

	output := buf.String()
	if !strings.Contains(output, "service") || !strings.Contains(output, "api") {
		t.Errorf("expected output to contain service and api, got: %s", output)
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	_ = xlog.Init(
		xlog.WithEnvironment(xlog.Development),
		xlog.WithLevel(slog.LevelWarn),
		xlog.WithOutput(&buf),
		xlog.WithSource(false),
	)

	ctx := context.Background()

	// Info should not be logged
	xlog.Info(ctx, "info message")
	if buf.Len() > 0 {
		t.Errorf("info should not be logged at warn level, got: %s", buf.String())
	}

	// Warn should be logged
	xlog.Warn(ctx, "warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Errorf("warn should be logged, got: %s", buf.String())
	}
}

func BenchmarkInfo(b *testing.B) {
	var buf bytes.Buffer
	_ = xlog.Init(
		xlog.WithEnvironment(xlog.Production),
		xlog.WithOutput(&buf),
		xlog.WithSource(false),
	)

	ctx := context.Background()
	ctx = xlog.WithTraceID(ctx, "trace-123")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		xlog.Info(ctx, "benchmark message", "iteration", i)
	}
}

func BenchmarkInfoParallel(b *testing.B) {
	var buf bytes.Buffer
	_ = xlog.Init(
		xlog.WithEnvironment(xlog.Production),
		xlog.WithOutput(&buf),
		xlog.WithSource(false),
	)

	ctx := context.Background()
	ctx = xlog.WithTraceID(ctx, "trace-123")

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			xlog.Info(ctx, "parallel benchmark", "iteration", i)
			i++
		}
	})
}
