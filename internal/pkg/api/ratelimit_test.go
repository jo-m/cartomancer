package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// noopHandler returns 200 OK without writing a body.
func noopHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimit_DisabledWhenZero(t *testing.T) {
	handler := rateLimit(0)(noopHandler())

	// Many consecutive requests must all succeed when the limit is disabled.
	for range 50 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimit_DisabledWhenNegative(t *testing.T) {
	handler := rateLimit(-1)(noopHandler())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_AllowsBurstThenRejects(t *testing.T) {
	// rps=5 produces a burst capacity of 5.
	handler := rateLimit(5)(noopHandler())

	// The first 5 requests fit the burst window and must pass.
	for i := range 5 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
		require.Equalf(t, http.StatusOK, rec.Code, "request %d should succeed", i)
	}

	// The next request, sent immediately, must be rejected with 429.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_FractionalRPSHasMinBurstOne(t *testing.T) {
	// rps below 1 must still allow at least one request through.
	handler := rateLimit(0.1)(noopHandler())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	// A second immediate request must be rejected.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}
