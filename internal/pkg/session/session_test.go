package session

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/app"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/password"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

func TestContext(t *testing.T) {
	ctx := t.Context()

	sess := db.Session{Uuid: "asdf"}
	ctx = withSession(ctx, sess)
	assert.Equal(t, sess, MustGet(ctx))

	user := db.User{Uuid: "asdf"}
	ctx = withUser(ctx, user)
	assert.Equal(t, user, MustGetUser(ctx))
}

const (
	userID     = "testid"
	userEmail  = "test@example.org"
	userPass   = "test"
	cookieName = "sid"
)

type testClient struct {
	t      *testing.T
	client *http.Client
	jar    *cookiejar.Jar
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	return &testClient{
		t:      t,
		client: &http.Client{Jar: jar},
		jar:    jar,
	}
}

func (c *testClient) doRequest(method, url string, body io.Reader, expectedStatus int) (string, []*http.Cookie, http.Header) {
	req, err := http.NewRequest(method, url, body)
	require.NoError(c.t, err)
	resp, err := c.client.Do(req)
	require.NoError(c.t, err)
	require.Equal(c.t, expectedStatus, resp.StatusCode)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)
	return string(respBody), resp.Cookies(), resp.Header
}

func createUser(t *testing.T, d *db.DB) {
	t.Helper()

	tx, err := d.BeginTX(t.Context())
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.CreateUser(t.Context(), db.CreateUserParams{
		Uuid:         userID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Email:        userEmail,
		Name:         "test",
		PasswordHash: utl.Must(password.Hash(userPass)),
		Admin:        0,
	})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	sess := MustGet(r.Context())
	fmt.Fprint(w, "session ", sess.Uuid)
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	user := GetUser(r.Context())
	if user != nil {
		fmt.Fprint(w, "user ", user.Uuid)
	} else {
		fmt.Fprint(w, "user nil")
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	sess := MustGet(r.Context())
	_, err := Create(r.Context(), sql.NullString{Valid: true, String: userID}, &sess)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	sess := MustGet(r.Context())
	err := Delete(r.Context(), &sess)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func assertSessionsCount(t *testing.T, d *db.DB, expected int64) {
	t.Helper()

	c, err := d.QueryRO().GetSessionsCount(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, expected, c)
}

func TestSessionMiddleware(t *testing.T) {
	// Setup database and create user.
	d := db.GetTestDB(t)
	defer d.Close()
	createUser(t, d)

	// Setup session store.
	conf := SessionConfig{
		IdleTimeout:             time.Second * 10,
		AbsoluteTimeout:         time.Second * 10,
		CookieName:              cookieName,
		insecureUseOnlyForTests: true,
	}
	appConf := app.AppConfig{
		AppName: "testapp",
	}
	sessionStore, err := NewStore(d, conf, appConf)
	require.NoError(t, err)

	// Setup test server.
	mux := http.NewServeMux()
	mux.Handle("/session", http.HandlerFunc(sessionHandler))
	mux.Handle("/user", http.HandlerFunc(userHandler))
	mux.Handle("/login", http.HandlerFunc(loginHandler))
	mux.Handle("/logout", http.HandlerFunc(logoutHandler))
	ts := httptest.NewServer(sessionStore.Middleware(mux))
	defer ts.Close()
	client := newTestClient(t)

	// Create the first session.
	body, cookies, _ := client.doRequest(http.MethodGet, ts.URL+"/user", nil, http.StatusOK)
	assert.Len(t, cookies, 1)
	assert.Equal(t, "user nil", body)
	assertSessionsCount(t, d, 1)

	// Same session.
	body0, cookies, _ := client.doRequest(http.MethodGet, ts.URL+"/session", nil, http.StatusOK)
	assert.Empty(t, cookies)
	assertSessionsCount(t, d, 1)
	body1, _, _ := client.doRequest(http.MethodGet, ts.URL+"/session", nil, http.StatusOK)
	assert.Equal(t, body0, body1)

	// Login.
	client.doRequest(http.MethodPost, ts.URL+"/login", nil, http.StatusNoContent)
	assertSessionsCount(t, d, 1)
	body2, _, _ := client.doRequest(http.MethodGet, ts.URL+"/session", nil, http.StatusOK)
	assert.NotEqual(t, body0, body2)
	body3, _, _ := client.doRequest(http.MethodGet, ts.URL+"/user", nil, http.StatusOK)
	assert.Equal(t, "user "+userID, body3)

	// Logout.
	_, cookies, _ = client.doRequest(http.MethodPost, ts.URL+"/logout", nil, http.StatusNoContent)
	assertSessionsCount(t, d, 0)
	body4, _, _ := client.doRequest(http.MethodGet, ts.URL+"/session", nil, http.StatusOK)
	assert.NotEqual(t, body2, body4)
	// Cookie was deleted.
	assert.Len(t, cookies, 1)
	assert.Equal(t, cookieName, cookies[0].Name)
	assert.Equal(t, "", cookies[0].Value)
}

func createSession(t *testing.T, d *db.DB, store *Store) string {
	t.Helper()

	tx, err := d.BeginTX(t.Context())
	require.NoError(t, err)
	defer tx.Rollback()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err = store.create(w, r, tx, sql.NullString{}, nil)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	cookieVal := w.Result().Cookies()[0].Value
	return cookieVal
}

func pokeSession(t *testing.T, d *db.DB, store *Store, cookieVal string) error {
	t.Helper()

	tx, err := d.BeginTX(t.Context())
	require.NoError(t, err)
	defer tx.Rollback()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{
		Value: cookieVal,
		Name:  cookieName,
	})
	_, err = store.get(r, tx)
	txErr := tx.Commit()
	require.NoError(t, txErr)

	return err
}

func TestSessionExpiry(t *testing.T) {
	// Setup database and create user.
	d := db.GetTestDB(t)
	defer d.Close()
	createUser(t, d)

	// Setup session store.
	conf := SessionConfig{
		IdleTimeout:             time.Millisecond * 100,
		AbsoluteTimeout:         time.Millisecond * 400,
		CookieName:              cookieName,
		insecureUseOnlyForTests: true,
	}
	appConf := app.AppConfig{
		AppName: "testapp",
	}
	store, err := NewStore(d, conf, appConf)
	require.NoError(t, err)
	require.Len(t, store.c.JWTSecret, jwtSecretLenBytes)

	// Create a session.
	cookieVal := createSession(t, d, store)
	// And it remains valid.
	for range 35 {
		require.NoError(t, pokeSession(t, d, store, cookieVal))
		time.Sleep(time.Millisecond * 10)
	}
	time.Sleep(time.Millisecond * 100)
	// Hit the absolute timeout.
	assert.EqualError(t, pokeSession(t, d, store, cookieVal), "session expired (absolute)")

	// Cleanup.
	assertSessionsCount(t, d, 1)
	store.cleanup(t.Context(), time.Now())
	assertSessionsCount(t, d, 0)

	// Create a new session.
	cookieVal = createSession(t, d, store)
	time.Sleep(time.Millisecond * 101)
	// Hit the idle timeout.
	assert.EqualError(t, pokeSession(t, d, store, cookieVal), "session expired (idle)")

	// Cleanup.
	assertSessionsCount(t, d, 1)
	store.cleanup(t.Context(), time.Now())
	assertSessionsCount(t, d, 0)
}
