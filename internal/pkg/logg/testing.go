package logg

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

// tbWriter is an [io.Writer] that forwards each written line to [testing.TB.Log].
type tbWriter struct {
	t testing.TB
}

// Write implements [io.Writer], forwarding p to t.Log after stripping the trailing newline.
func (w *tbWriter) Write(p []byte) (n int, err error) {
	w.t.Helper()
	w.t.Log(string(bytes.TrimRight(p, "\n")))
	return len(p), nil
}

// NewTestHandler returns an [slog.Handler] that forwards log records to t.Log.
// All levels down to [LevelTrace] are captured.
func NewTestHandler(t testing.TB) slog.Handler {
	t.Helper()
	return NewHandler(LoggConfig{LogPretty: true, LogLevel: LevelTrace}, &tbWriter{t: t})
}

// WithTestLogger attaches a logger backed by t.Log to the context.
// All levels down to [LevelTrace] are captured.
func WithTestLogger(ctx context.Context, t testing.TB) context.Context {
	t.Helper()
	return WithLogger(ctx, slog.New(NewTestHandler(t)))
}
