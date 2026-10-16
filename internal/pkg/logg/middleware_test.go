package logg

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPanickingTestHandler returns a router mirroring the middleware stack of the
// main server, with a handler which panics.
func newPanickingTestHandler(logs *bytes.Buffer) http.Handler {
	logger := slog.New(NewHandler(LoggConfig{LogPretty: false}, logs))

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(AttachLogger(logger))
	mux.Use(RequestLogger)
	mux.Use(middleware.Recoverer)
	mux.Get("/boom", func(http.ResponseWriter, *http.Request) {
		panic("kaboom")
	})
	return mux
}

func TestRequestLoggerReportsRecoveredPanic(t *testing.T) {
	logs := bytes.Buffer{}
	mux := newPanickingTestHandler(&logs)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

	// The panic is recovered by chi's Recoverer.
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	out := logs.String()
	assert.Contains(t, out, `"level":"ERROR"`)
	assert.Contains(t, out, `"msg":"recovered panic in request handler"`)
	assert.Contains(t, out, `"panic":"kaboom"`)
	// Stack trace of the panicking goroutine.
	assert.Contains(t, out, `"stack":"goroutine `)
	// The panic report carries the request ID of the failing request.
	assert.Contains(t, out, `"reqID":"`)
	// It precedes the access log entry, which reports the status written by the
	// recoverer.
	panicAt := strings.Index(out, "recovered panic in request handler")
	statusAt := strings.Index(out, `"status":500`)
	require.NotEqual(t, -1, panicAt)
	require.NotEqual(t, -1, statusAt)
	assert.Less(t, panicAt, statusAt)
}

// newStatusTestHandler returns a router mirroring the middleware stack of the
// main server, with a handler which responds with status.
func newStatusTestHandler(logs *bytes.Buffer, status int) http.Handler {
	logger := slog.New(NewHandler(LoggConfig{LogPretty: false, LogLevel: LevelTrace}, logs))

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(AttachLogger(logger))
	mux.Use(RequestLogger)
	mux.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	})
	return mux
}

// TestRequestLoggerCanceledRequestLogsDebug checks that a request aborted by
// the client is not logged as an error, even if the handler responded with a
// server error status before noticing the disconnect.
func TestRequestLoggerCanceledRequestLogsDebug(t *testing.T) {
	logs := bytes.Buffer{}
	mux := newStatusTestHandler(&logs, http.StatusInternalServerError)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	out := logs.String()
	assert.Contains(t, out, `"level":"DEBUG"`)
	assert.NotContains(t, out, `"level":"ERROR"`)
}

func TestRequestLoggerServerErrorLogsError(t *testing.T) {
	logs := bytes.Buffer{}
	mux := newStatusTestHandler(&logs, http.StatusInternalServerError)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logs.String(), `"level":"ERROR"`)
}
