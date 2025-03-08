package logg

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// AttachLogger attaches a logger with the request ID attribute to the request context.
func AttachLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			reqId := middleware.GetReqID(ctx)
			reqLogger := logger.With("reqId", reqId)
			ctx = WithLogger(ctx, reqLogger)
			h.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
	return f
}

func getLevel(code int) slog.Level {
	if code >= 500 {
		return slog.LevelError
	}
	if code >= 400 {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

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
			Log(r.Context(), getLevel(ww.Status()), msg, "url", r.URL, "method", r.Method, "status", ww.Status(), "duration", time.Since(t0))
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}
