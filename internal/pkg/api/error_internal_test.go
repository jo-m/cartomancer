package api

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

// newTestLogBuffer returns a context with a logger writing JSON records to the
// returned buffer, and the buffer itself.
func newTestLogBuffer() (context.Context, *bytes.Buffer) {
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: logg.LevelTrace}))
	return logg.WithLogger(context.Background(), logger), logs
}

func TestWriteServerErrorLogsAndWrites500(t *testing.T) {
	ctx, logs := newTestLogBuffer()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	writeServerError(rec, req, "failed to do thing", errors.New("boom"), "k", "v")

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.JSONEq(t, `{"msg":"internal server error"}`, rec.Body.String())
	require.Contains(t, logs.String(), `"level":"ERROR"`)
	require.Contains(t, logs.String(), "failed to do thing")
	require.Contains(t, logs.String(), "boom")
	require.Contains(t, logs.String(), `"k":"v"`)
}

func TestWriteServerErrorSkipsCanceledClient(t *testing.T) {
	ctx, logs := newTestLogBuffer()
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	// The wrapped recorder reports status 0 as long as nothing is written.
	ww := middleware.NewWrapResponseWriter(httptest.NewRecorder(), 1)

	writeServerError(ww, req, "failed to do thing", errors.New("context canceled"))

	require.Zero(t, ww.Status())
	require.Contains(t, logs.String(), `"level":"DEBUG"`)
	require.NotContains(t, logs.String(), `"level":"ERROR"`)
}
