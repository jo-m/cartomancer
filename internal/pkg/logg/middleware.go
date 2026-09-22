package logg

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// AttachLogger is a net/http middleware which attaches a logger with the request ID attribute to the request context.
// Use [GetLogger] to retrieve it.
func AttachLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			reqID := middleware.GetReqID(ctx)
			reqLogger := logger.With("reqID", reqID)
			ctx = WithLogger(ctx, reqLogger)
			h.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
	return f
}

func levelFor(statusCode int) slog.Level {
	if statusCode >= 500 {
		return slog.LevelError
	}
	if statusCode >= 400 {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

// logFormatter creates the [middleware.LogEntry] attached to each request by
// [RequestLogger]. It implements [middleware.LogFormatter].
type logFormatter struct{}

// NewLogEntry implements [middleware.LogFormatter].
func (logFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &logEntry{ctx: r.Context()}
}

// logEntry implements [middleware.LogEntry], through which chi's
// [middleware.Recoverer] reports panics recovered from handlers. It logs via
// the logger attached by [AttachLogger], so panic reports carry the request ID.
type logEntry struct {
	ctx context.Context
}

// Write implements [middleware.LogEntry]. It is never called: access logging is
// done by [RequestLogger] directly, as chi's own request logger is not used.
func (e *logEntry) Write(_ int, _ int, _ http.Header, _ time.Duration, _ any) {
}

// Panic implements [middleware.LogEntry]. It is called by chi's
// [middleware.Recoverer] with the value recovered from the panicking handler
// and the stack of the goroutine it panicked in.
func (e *logEntry) Panic(v any, stack []byte) {
	Error(e.ctx, "recovered panic in request handler", "panic", v, "stack", string(stack))
}

// RequestLogger is a net/http middleware which logs each request.
// It expects the [AttachLogger] middleware above in the stack.
//
// It also attaches a [middleware.LogEntry] to the request context, through
// which a [middleware.Recoverer] further down the stack reports recovered
// panics.
func RequestLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		t0 := time.Now()
		defer func() {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			url := fmt.Sprintf("%s %s://%s%s", r.Proto, scheme, r.Host, r.RequestURI)
			msg := fmt.Sprintf("%s %s %d", r.Method, url, ww.Status())
			Log(r.Context(), levelFor(ww.Status()), msg, "url", r.URL, "method", r.Method, "status", ww.Status(), "duration", time.Since(t0))
		}()

		next.ServeHTTP(ww, middleware.WithLogEntry(r, logFormatter{}.NewLogEntry(r)))
	}
	return http.HandlerFunc(fn)
}
