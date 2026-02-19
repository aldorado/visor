package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
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
	args := make([]any, 0, len(attrs)+6)
	args = append(args, "component", l.component)
	args = append(args, "function", caller(3))
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		args = append(args, "request_id", requestID)
	}
	args = append(args, attrs...)
	l.base.Log(ctx, level, msg, args...)
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
