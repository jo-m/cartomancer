package rest_test

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminListUsers_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	var resp []map[string]any
	status, _ := e.do(client, http.MethodGet, "/admin/users", nil, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Len(t, resp, 2)
}

func TestAdminListUsers_Forbidden(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodGet, "/admin/users", nil, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestAdminListUsers_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodGet, "/admin/users", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestAdminCreateUser_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	var resp map[string]any
	status, _ := e.do(client, http.MethodPost, "/admin/users", map[string]any{
		"email": "new@example.com",
		"name":  "New-User",
		"admin": false,
	}, &resp)
	assert.Equal(t, http.StatusCreated, status)
	assert.Equal(t, "new@example.com", resp["email"])
	assert.NotEmpty(t, resp["initialPassword"])
}

func TestAdminCreateUser_MissingEmail(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodPost, "/admin/users", map[string]any{
		"name": "New-User",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestAdminCreateUser_MissingName(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodPost, "/admin/users", map[string]any{
		"email": "new@example.com",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestAdminGetUser_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	uuid := e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	var resp map[string]any
	status, _ := e.do(client, http.MethodGet, "/admin/users/"+uuid, nil, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "alice@example.com", resp["email"])
}

func TestAdminGetUser_NotFound(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodGet, "/admin/users/nonexistent-uuid", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestAdminUpdateUser_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	uuid := e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	var resp map[string]any
	status, _ := e.do(client, http.MethodPatch, "/admin/users/"+uuid, map[string]any{
		"email": "alice2@example.com",
		"name":  "Alice-Two",
		"admin": false,
	}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "alice2@example.com", resp["email"])
	assert.Equal(t, "Alice-Two", resp["name"])
}

func TestAdminUpdateUser_NotFound(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodPatch, "/admin/users/no-such-uuid", map[string]any{
		"email": "x@example.com",
		"name":  "Xxx",
		"admin": false,
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestAdminDeleteUser_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	uuid := e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodDelete, "/admin/users/"+uuid, nil, nil)
	assert.Equal(t, http.StatusNoContent, status)

	// Confirm the user is gone.
	status, _ = e.do(client, http.MethodGet, "/admin/users/"+uuid, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestAdminDeleteUser_NotFound(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodDelete, "/admin/users/no-such-uuid", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestAdminResetPassword_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	uuid := e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	var resp map[string]any
	status, _ := e.do(client, http.MethodPost, "/admin/users/"+uuid+"/reset-password", map[string]any{
		"sendEmail": false,
	}, &resp)
	assert.Equal(t, http.StatusOK, status)
	newPass, ok := resp["password"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, newPass)

	// Old password no longer works.
	client2 := e.newClient()
	status2, _ := e.do(client2, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice@example.com",
		"password": "secret",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status2)

	// New password works.
	client3 := e.newClient()
	status3, _ := e.do(client3, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice@example.com",
		"password": newPass,
	}, nil)
	assert.Equal(t, http.StatusOK, status3)
}

func TestAdminResetPassword_InvalidatesSessions(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	uuid := e.createUser("alice@example.com", "Alice", "secret", false)

	// Use isolated transports to avoid HTTP/2 cookie leaking between clients.
	newIsolatedClient := func() *http.Client {
		jar, err := cookiejar.New(nil)
		require.NoError(t, err)
		return &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
			},
			Jar: jar,
		}
	}

	// Alice logs in.
	aliceClient := newIsolatedClient()
	e.login(aliceClient, "alice@example.com", "secret")

	// Verify Alice's session works.
	status, _ := e.do(aliceClient, http.MethodGet, "/tracks/editing", nil, nil)
	require.Equal(t, http.StatusOK, status)

	// Admin resets Alice's password.
	adminClient := newIsolatedClient()
	e.login(adminClient, "admin@example.com", "adminpass")
	status, _ = e.do(adminClient, http.MethodPost, "/admin/users/"+uuid+"/reset-password", map[string]any{
		"sendEmail": false,
	}, nil)
	require.Equal(t, http.StatusOK, status)

	// Alice's session is invalidated.
	status, _ = e.do(aliceClient, http.MethodGet, "/tracks/editing", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestAdminResetPassword_NotFound(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodPost, "/admin/users/no-such-uuid/reset-password", map[string]any{
		"sendEmail": false,
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestAdminCreateUser_DuplicateEmail(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	e.createUser("existing@example.com", "Existing", "secret", false)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodPost, "/admin/users", map[string]any{
		"email": "existing@example.com",
		"name":  "New-User",
		"admin": false,
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestAdminCreateUser_DuplicateName(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	e.createUser("existing@example.com", "Existing", "secret", false)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodPost, "/admin/users", map[string]any{
		"email": "new@example.com",
		"name":  "Existing",
		"admin": false,
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestAdminCreateUser_AdminFlag(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	var resp map[string]any
	status, _ := e.do(client, http.MethodPost, "/admin/users", map[string]any{
		"email": "newadmin@example.com",
		"name":  "New-Admin",
		"admin": true,
	}, &resp)
	assert.Equal(t, http.StatusCreated, status)
	assert.Equal(t, true, resp["admin"])
}

func TestAdminCreateUser_InvalidName(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodPost, "/admin/users", map[string]any{
		"email": "new@example.com",
		"name":  "ab",
		"admin": false,
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestAdminUpdateUser_EmailTaken(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	uuid := e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret", false)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodPatch, "/admin/users/"+uuid, map[string]any{
		"email": "bob@example.com",
		"name":  "Alice",
		"admin": false,
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestAdminUpdateUser_NameTaken(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	uuid := e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret", false)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodPatch, "/admin/users/"+uuid, map[string]any{
		"email": "alice@example.com",
		"name":  "Bob",
		"admin": false,
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestAdminUpdateUser_ToggleAdmin(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	uuid := e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	// Promote to admin.
	var resp map[string]any
	status, _ := e.do(client, http.MethodPatch, "/admin/users/"+uuid, map[string]any{
		"email": "alice@example.com",
		"name":  "Alice",
		"admin": true,
	}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, true, resp["admin"])

	// Remove admin status.
	status, _ = e.do(client, http.MethodPatch, "/admin/users/"+uuid, map[string]any{
		"email": "alice@example.com",
		"name":  "Alice",
		"admin": false,
	}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, false, resp["admin"])
}

func TestAdminUpdateUser_EmailChange_OldEmailInvalidated(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	uuid := e.createUser("alice@example.com", "Alice", "secret", false)
	adminClient := e.newClient()
	e.login(adminClient, "admin@example.com", "adminpass")

	status, _ := e.do(adminClient, http.MethodPatch, "/admin/users/"+uuid, map[string]any{
		"email": "alice-new@example.com",
		"name":  "Alice",
		"admin": false,
	}, nil)
	require.Equal(t, http.StatusOK, status)

	// Old email no longer works for login.
	loginClient := e.newClient()
	status, _ = e.do(loginClient, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice@example.com",
		"password": "secret",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)

	// New email works for login.
	status, _ = e.do(loginClient, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice-new@example.com",
		"password": "secret",
	}, nil)
	assert.Equal(t, http.StatusOK, status)
}
