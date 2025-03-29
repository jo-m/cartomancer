package logg

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// AttachLogger is a net/http middleware which attaches a logger with the request ID attribute to the request context.
// Use GetLogger() to retrieve it.
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

// RequestLogger is a net/http middleware which logs each request.
// It expects the AttachLogger() middleware above in the stack.
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

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}
