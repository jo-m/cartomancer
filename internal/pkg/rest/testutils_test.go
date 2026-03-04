package rest_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/app"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/password"
	"jo-m.ch/go/detour/internal/pkg/rest"
	"jo-m.ch/go/detour/internal/pkg/session"
)

const testCookieName = "sid"

// tWriter bridges slog output to testing.T so log lines show up
// alongside test output and are captured on failure.
type tWriter struct{ t *testing.T }

func (w *tWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

// testEnv holds the test server and database for a single test.
type testEnv struct {
	t  *testing.T
	d  *db.DB
	ts *httptest.Server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	logger := slog.New(logg.NewHandler(logg.LoggConfig{
		LogPretty: false,
		LogLevel:  slog.LevelDebug,
	}, &tWriter{t: t}))

	workers, err := jobs.NewWorkers(logg.WithLogger(t.Context(), logger), d, jobs.JobsConfig{MaxParallel: 1})
	require.NoError(t, err)

	sessConf := session.SessionConfig{
		IdleTimeout:     time.Hour,
		AbsoluteTimeout: time.Hour,
		CookieName:      testCookieName,
		CookiePath:      "/",
	}
	sess, err := session.NewStore(d, sessConf, app.AppConfig{AppName: "test"})
	require.NoError(t, err)

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(logg.AttachLogger(logger))
	mux.Use(sess.Middleware)
	mux.Mount("/", rest.New(d, sess, workers.Submitter(), app.AppConfig{AppName: "test"}))

	ts := httptest.NewTLSServer(mux)
	t.Cleanup(ts.Close)

	return &testEnv{t: t, d: d, ts: ts}
}

// newClient creates a new TLS-aware HTTP client with an empty cookie jar.
func (e *testEnv) newClient() *http.Client {
	e.t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(e.t, err)
	c := e.ts.Client()
	c.Jar = jar
	return c
}

// createUser inserts a user directly into the DB and returns its UUID.
func (e *testEnv) createUser(email, name, pass string, admin bool) string {
	e.t.Helper()

	var adminVal int64
	if admin {
		adminVal = 1
	}

	id, err := uuid.NewV7()
	require.NoError(e.t, err)

	now := time.Now().UTC()
	hash, err := password.Hash(pass)
	require.NoError(e.t, err)

	u, err := e.d.QueryRW().CreateUser(e.t.Context(), db.CreateUserParams{
		Uuid:           id.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
		Email:          email,
		Name:           name,
		PasswordHash:   hash,
		Admin:          adminVal,
		EmailConfirmed: 1,
	})
	require.NoError(e.t, err)
	return u.Uuid
}

// login performs POST /sessions/login with the given client and asserts success.
func (e *testEnv) login(client *http.Client, email, pass string) {
	e.t.Helper()
	status, _ := e.do(client, http.MethodPost, "/sessions/login", map[string]string{
		"email":    email,
		"password": pass,
	}, nil)
	require.Equal(e.t, http.StatusOK, status)
}

// getVerificationToken retrieves the email verification token for a given email directly from the DB.
func (e *testEnv) getVerificationToken(t *testing.T, email string) string {
	t.Helper()
	ver, err := e.d.QueryRO().GetEmailVerificationByEmail(t.Context(), email)
	require.NoError(t, err, "no verification token found for %s", email)
	return ver.Token
}

// do sends a JSON request and returns the status code and raw response body.
// If result is non-nil, the response body is JSON-decoded into it.
func (e *testEnv) do(client *http.Client, method, path string, body any, result any) (int, []byte) {
	e.t.Helper()

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(e.t, err)
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, e.ts.URL+path, reqBody)
	require.NoError(e.t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	require.NoError(e.t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)

	if result != nil {
		require.NoError(e.t, json.Unmarshal(raw, result))
	}

	return resp.StatusCode, raw
}
