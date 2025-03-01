package logging

import (
	"context"
	"log/slog"
	"os"
)

type loggerKey struct{}

// NewLogger creates a new slog.Logger with the given options.
func NewLogger(opts *slog.HandlerOptions) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

// WithLogger returns a new context with the given logger attached.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// GetLogger retrieves the logger from the context. If no logger is found,
// it returns a default logger.
func GetLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	// Default logger if none is in context
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// Log logs a message with the given level and attributes using the logger from the context.
func Log(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	GetLogger(ctx).Log(ctx, level, msg, attrs...)
}

// Info logs an informational message.
func Info(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelInfo, msg, attrs...)
}

// Warn logs a warning message.
func Warn(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelWarn, msg, attrs...)
}

// Error logs an error message.
func Error(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelError, msg, attrs...)
}

// Debug logs a debug message.
func Debug(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelDebug, msg, attrs...)
}
