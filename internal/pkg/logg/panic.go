package logg

import (
	"context"
	"errors"
	"log/slog"
)

type panicHandler struct{}

// ErrNotAllowed is [panic()] message when the default logger is used when [DisableDefaultLogger] was called before.
var ErrNotAllowed = errors.New("using the default logger is not allowed")

// Enabled implements slog.Handler.
func (p *panicHandler) Enabled(context.Context, slog.Level) bool {
	panic(ErrNotAllowed)
}

// Handle implements slog.Handler.
func (p *panicHandler) Handle(context.Context, slog.Record) error {
	panic(ErrNotAllowed)
}

// WithAttrs implements slog.Handler.
func (p *panicHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	panic(ErrNotAllowed)
}

// WithGroup implements slog.Handler.
func (p *panicHandler) WithGroup(name string) slog.Handler {
	panic(ErrNotAllowed)
}

var _ slog.Handler = (*panicHandler)(nil)

// DisableDefaultLogger sets the slog default logger to a logger which will panic on any invocation.
// This is useful to ensure that all log messages are sent through the context logger instead.
// Should be used only in development and testing environments.
func DisableDefaultLogger() {
	panicLogger := slog.New(&panicHandler{})
	slog.SetDefault(panicLogger)
}
