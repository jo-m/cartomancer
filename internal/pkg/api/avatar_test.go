package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserAvatar_Success(t *testing.T) {
	e := newTestEnv(t)
	uuid := e.createUser("avatar@example.com", "AvatarUser", "secret11", false)

	client := e.newClient()
	status, body := e.do(client, http.MethodGet, "/users/"+uuid+"/avatar", nil, nil)

	assert.Equal(t, http.StatusOK, status)
	assert.True(t, strings.Contains(string(body), "<svg"), "expected SVG response")
}

func TestGetUserAvatar_NotFound(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodGet, "/users/nonexistent-uuid/avatar", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestGetUserAvatar_NoAuthRequired(t *testing.T) {
	e := newTestEnv(t)
	uuid := e.createUser("pub@example.com", "PubUser", "secret11", false)

	// Unauthenticated client (fresh, no login).
	client := e.newClient()
	status, body := e.do(client, http.MethodGet, "/users/"+uuid+"/avatar", nil, nil)

	assert.Equal(t, http.StatusOK, status)
	assert.True(t, strings.Contains(string(body), "<svg"), "expected SVG response")
}

func TestRotateAvatar_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("rotate@example.com", "RotateUser", "secret11", false)
	client := e.newClient()
	e.login(client, "rotate@example.com", "secret11")

	// Get initial seed.
	var before map[string]any
	status, _ := e.do(client, http.MethodGet, "/sessions", nil, &before)
	require.Equal(t, http.StatusOK, status)
	beforeUser := before["user"].(map[string]any)
	oldSeed := beforeUser["avatarSeed"].(string)
	require.NotEmpty(t, oldSeed)

	// Rotate.
	var rotated map[string]any
	status, _ = e.do(client, http.MethodPost, "/account/rotate-avatar", nil, &rotated)
	assert.Equal(t, http.StatusOK, status)

	newSeed, ok := rotated["avatarSeed"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, newSeed)
	assert.NotEqual(t, oldSeed, newSeed)
}

func TestRotateAvatar_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/account/rotate-avatar", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}
