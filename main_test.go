package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/app"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/db/forecastdb"
	"jo-m.ch/go/cartomancer/internal/pkg/db/geonamesdb"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/password"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
)

func TestEnsureInitialAdmin_CreatesUser(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := t.Context()

	created, pass, err := ensureInitialAdmin(ctx, d, "admin@example.com", "")
	require.NoError(t, err)
	require.True(t, created)
	require.NotEmpty(t, pass)

	// Verify the user exists and the password works.
	user, err := d.QueryRO().GetUserByEmail(ctx, "admin@example.com")
	require.NoError(t, err)
	require.Equal(t, "Admin", user.Name)
	require.Equal(t, int64(1), user.Admin)
	require.Equal(t, int64(1), user.EmailConfirmed)
	require.True(t, password.Check(pass, user.PasswordHash))
}

func TestEnsureInitialAdmin_ExplicitPassword(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := t.Context()

	created, pass, err := ensureInitialAdmin(ctx, d, "admin@example.com", "my-known-pass")
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "my-known-pass", pass)

	user, err := d.QueryRO().GetUserByEmail(ctx, "admin@example.com")
	require.NoError(t, err)
	require.True(t, password.Check("my-known-pass", user.PasswordHash))
}

func TestEnsureInitialAdmin_Idempotent(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := t.Context()

	created1, pass1, err := ensureInitialAdmin(ctx, d, "admin@example.com", "")
	require.NoError(t, err)
	require.True(t, created1)
	require.NotEmpty(t, pass1)

	// Second call with the same email does nothing.
	created2, pass2, err := ensureInitialAdmin(ctx, d, "admin@example.com", "")
	require.NoError(t, err)
	require.False(t, created2)
	require.Empty(t, pass2)

	// Original password still works.
	user, err := d.QueryRO().GetUserByEmail(ctx, "admin@example.com")
	require.NoError(t, err)
	require.True(t, password.Check(pass1, user.PasswordHash))
}

// TestNonAPIRoutesIgnoreSessions asserts that the session middleware only runs on
// API routes. Once both connection pools are closed, a request carrying a valid
// session cookie still succeeds for the SPA and for /robots.txt, while the same
// cookie on an API route makes the request fail, as there the session must be
// resolved against the database.
func TestNonAPIRoutesIgnoreSessions(t *testing.T) {
	d := db.GetTestDB(t)
	gd := geonamesdb.GetTestDB(t)
	fd := forecastdb.GetTestDB(t)

	ctx := logg.WithTestLogger(t.Context(), t)
	workers, err := jobs.NewWorkers(ctx, d, jobs.JobsConfig{MaxParallel: 1})
	require.NoError(t, err)

	// Minimal stand-in for the frontend build output, so the test does not require
	// a populated static/ directory.
	staticFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>test</title>")},
	}

	h := newHandler(ctx, d, gd, fd, session.SessionConfig{
		IdleTimeout:     time.Hour,
		AbsoluteTimeout: time.Hour,
		CookieName:      "sid",
		CookiePath:      "/",
	}, app.AppConfig{InstanceName: "test"}, staticFS, workers.Submitter(), 0, 0, t.TempDir())
	ts := httptest.NewTLSServer(h)
	defer ts.Close()

	// Log in over the API to get a valid session cookie into the jar.
	userID, err := uuid.NewV7()
	require.NoError(t, err)
	hash, err := password.Hash("password")
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = d.QueryRW().CreateUser(ctx, db.CreateUserParams{
		Uuid:           userID.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
		Email:          "user@example.com",
		Name:           "User",
		PasswordHash:   hash,
		EmailConfirmed: 1,
	})
	require.NoError(t, err)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := ts.Client()
	client.Jar = jar

	loginBody, err := json.Marshal(map[string]string{"email": "user@example.com", "password": "password"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/sessions/login", bytes.NewReader(loginBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "cartomancer")
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// No database access is possible from here on.
	require.NoError(t, d.Close())

	for _, path := range []string{"/robots.txt", "/", "/index.html"} {
		t.Run(path, func(t *testing.T) {
			resp, err := client.Get(ts.URL + path)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}

	// Control: on an API route the same client does resolve its session, which the
	// closed database turns into an internal server error. This also shows that the
	// requests above really carried the cookie.
	resp, err = client.Get(ts.URL + "/api/sessions")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
