// Package logg deals with logging.
package logg

import (
	"context"
	"log/slog"
)

const (
	// LevelTrace is below LevelDebug.
	LevelTrace = slog.Level(-8)
	// LevelPanic is above LevelError, and additionally terminates the program.
	LevelPanic = slog.Level(10)
)

type ctxKey struct{}

// WithLogger attaches a logger to the context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// WithDiscardHandler returns a context with a logger that discards all logs.
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

// Log forwards to the logger in the context.
func Log(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	GetLogger(ctx).Log(ctx, level, msg, attrs...)
}

// Info forwards to the logger in the context.
func Info(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelInfo, msg, attrs...)
}

// Warn forwards to the logger in the context.
func Warn(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelWarn, msg, attrs...)
}

// Error forwards to the logger in the context.
func Error(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelError, msg, attrs...)
}

// Debug forwards to the logger in the context.
func Debug(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelDebug, msg, attrs...)
}

// Trace forwards to the logger in the context.
func Trace(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, LevelTrace, msg, attrs...)
}

// Panic forwards to the logger in the context and panics.
func Panic(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, LevelPanic, msg, attrs...)
	panic(msg)
}
