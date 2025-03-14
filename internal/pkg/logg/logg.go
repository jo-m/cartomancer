// Package logg deals with logging.
package logg

import (
	"context"
	"log/slog"
)

const (
	LevelTrace = slog.Level(-8)
	LevelPanic = slog.Level(10)
)

type ctxKey struct{}

// WithLogger attaches a logger to the context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

func WithDiscardHandler(ctx context.Context) context.Context {
	logger := slog.New(slog.DiscardHandler)
	return WithLogger(ctx, logger)
}

// GetLogger retrieves a logger from the context.
func GetLogger(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
			return logger
		}
	}
	return slog.Default()
}

func Log(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	GetLogger(ctx).Log(ctx, level, msg, attrs...)
}

func Info(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelInfo, msg, attrs...)
}

func Warn(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelWarn, msg, attrs...)
}

func Error(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelError, msg, attrs...)
}

func Debug(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelDebug, msg, attrs...)
}

func Trace(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, LevelTrace, msg, attrs...)
}

func Panic(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, LevelPanic, msg, attrs...)
	panic(msg)
}
