// Package logg deals with logging.
package logg

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

const (
	// LevelTrace is below [slog.LevelDebug].
	LevelTrace = slog.Level(-8)
	// LevelPanic is above [slog.LevelError], and additionally terminates the program.
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

// Copied and adapted from the stdlib slog package.
func log(ctx context.Context, logger *slog.Logger, level slog.Level, msg string, args ...any) {
	if !logger.Enabled(ctx, level) {
		return
	}
	var pc uintptr
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.Add(args...)
	if ctx == nil {
		ctx = context.Background()
	}
	_ = logger.Handler().Handle(ctx, r)

	if level == LevelPanic {
		panic(msg)
	}
}

// Log logs via the logger in the context.
func Log(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	log(ctx, GetLogger(ctx), level, msg, attrs...)
}

// Trace logs via the logger in the context.
func Trace(ctx context.Context, msg string, attrs ...any) {
	log(ctx, GetLogger(ctx), LevelTrace, msg, attrs...)
}

// Debug logs via the logger in the context.
func Debug(ctx context.Context, msg string, attrs ...any) {
	log(ctx, GetLogger(ctx), slog.LevelDebug, msg, attrs...)
}

// Info logs via the logger in the context.
func Info(ctx context.Context, msg string, attrs ...any) {
	log(ctx, GetLogger(ctx), slog.LevelInfo, msg, attrs...)
}

// Warn logs via the logger in the context.
func Warn(ctx context.Context, msg string, attrs ...any) {
	log(ctx, GetLogger(ctx), slog.LevelWarn, msg, attrs...)
}

// Error logs via the logger in the context.
func Error(ctx context.Context, msg string, attrs ...any) {
	log(ctx, GetLogger(ctx), slog.LevelError, msg, attrs...)
}

// Err logs via the logger in the context.
func Err(ctx context.Context, msg string, err error, attrs ...any) {
	attrs = append(attrs, "err", err)
	log(ctx, GetLogger(ctx), slog.LevelError, msg, attrs...)
}

// Panic logs via the logger in the context and panics.
func Panic(ctx context.Context, msg string, attrs ...any) {
	log(ctx, GetLogger(ctx), LevelPanic, msg, attrs...)
	panic(msg)
}
