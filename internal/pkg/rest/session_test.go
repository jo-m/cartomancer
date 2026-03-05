package rest_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSession_Anonymous(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	var resp map[string]any
	status, _ := e.do(client, http.MethodGet, "/sessions", nil, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.NotEmpty(t, resp["sessionUuid"])
	assert.Nil(t, resp["user"])
}

func TestLogin_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()

	var resp map[string]any
	status, _ := e.do(client, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice@example.com",
		"password": "secret",
	}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.NotEmpty(t, resp["sessionUuid"])
	user, ok := resp["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "alice@example.com", user["email"])
	assert.Equal(t, false, user["admin"])
}

func TestLogin_WrongPassword(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice@example.com",
		"password": "wrong",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestLogin_UnknownEmail(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "irrelevant",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestLogin_EmailNotConfirmed(t *testing.T) {
	e := newTestEnv(t)
	e.createUnconfirmedUser("alice@example.com", "Alice", "secret")
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice@example.com",
		"password": "secret",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestLogin_AlreadyLoggedIn(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice@example.com",
		"password": "secret",
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestGetSession_AfterLogin(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	var resp map[string]any
	status, _ := e.do(client, http.MethodGet, "/sessions", nil, &resp)
	assert.Equal(t, http.StatusOK, status)
	user, ok := resp["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "alice@example.com", user["email"])
}

func TestLogout_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodPost, "/sessions/logout", nil, nil)
	assert.Equal(t, http.StatusNoContent, status)

	// After logout the session has no user.
	var resp map[string]any
	status, _ = e.do(client, http.MethodGet, "/sessions", nil, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Nil(t, resp["user"])
}

func TestLogin_MixedCaseEmail(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "ALICE@EXAMPLE.COM",
		"password": "secret",
	}, nil)
	assert.Equal(t, http.StatusOK, status)
}

func TestLogin_EmptyFields(t *testing.T) {
	tests := []struct {
		desc string
		body map[string]string
	}{
		{"empty email", map[string]string{"email": "", "password": "secret"}},
		{"empty password", map[string]string{"email": "alice@example.com", "password": ""}},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			e := newTestEnv(t)
			e.createUser("alice@example.com", "Alice", "secret", false)
			client := e.newClient()
			status, _ := e.do(client, http.MethodPost, "/sessions/login", tc.body, nil)
			assert.Equal(t, http.StatusUnauthorized, status)
		})
	}
}

func TestLogout_Anonymous(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/sessions/logout", nil, nil)
	assert.Equal(t, http.StatusNoContent, status)
}
