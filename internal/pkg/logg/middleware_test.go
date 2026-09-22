package logg

import (
	"bytes"
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
