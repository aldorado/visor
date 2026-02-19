package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type LogConfig struct {
	Level   string
	Verbose bool
}

type Logger struct {
	base      *slog.Logger
	component string
}

func Init(cfg LogConfig) *slog.Logger {
	level := parseLevel(cfg.Level)
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.Verbose,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func Component(name string) *Logger {
	return &Logger{base: slog.Default(), component: name}
}

func (l *Logger) Debug(ctx context.Context, msg string, attrs ...any) {
	l.log(ctx, slog.LevelDebug, msg, attrs...)
}

func (l *Logger) Info(ctx context.Context, msg string, attrs ...any) {
	l.log(ctx, slog.LevelInfo, msg, attrs...)
}

func (l *Logger) Warn(ctx context.Context, msg string, attrs ...any) {
	l.log(ctx, slog.LevelWarn, msg, attrs...)
}

func (l *Logger) Error(ctx context.Context, msg string, attrs ...any) {
	l.log(ctx, slog.LevelError, msg, attrs...)
}

func (l *Logger) log(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	fn := caller(3)
	args := make([]any, 0, len(attrs)+10)
	args = append(args, "component", l.component)
	args = append(args, "function", fn)
	otelAttrs := []attribute.KeyValue{
		attribute.String("component", l.component),
		attribute.String("function", fn),
	}
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		args = append(args, "request_id", requestID)
		otelAttrs = append(otelAttrs, attribute.String("request_id", requestID))
	}

	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		args = append(args, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
		otelAttrs = append(otelAttrs,
			attribute.String("trace_id", sc.TraceID().String()),
			attribute.String("span_id", sc.SpanID().String()),
		)
	}

	args = append(args, attrs...)
	l.base.Log(ctx, level, msg, args...)

	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(msg, trace.WithAttributes(otelAttrs...))
	}
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func caller(depth int) string {
	pc := make([]uintptr, 1)
	n := runtime.Callers(depth, pc)
	if n == 0 {
		return "unknown"
	}
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	fn := frame.Function
	if fn == "" {
		return "unknown"
	}
	parts := strings.Split(fn, "/")
	return parts[len(parts)-1]
}

func AttrErr(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "")
	}
	return slog.String("error", err.Error())
}

func AttrAny(key string, value any) slog.Attr {
	return slog.Any(key, value)
}

func Must(err error, msg string) {
	if err != nil {
		slog.Error(msg, "error", err.Error())
		panic(fmt.Sprintf("%s: %v", msg, err))
	}
}
