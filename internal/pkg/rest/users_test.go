package rest_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAccount_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	var resp map[string]any
	status, _ := e.do(client, http.MethodPatch, "/account", map[string]string{"name": "Alice Updated"}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Alice Updated", resp["name"])
}

func TestUpdateAccount_MissingName(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodPatch, "/account", map[string]string{"name": ""}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateAccount_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPatch, "/account", map[string]string{"name": "X"}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestChangePassword_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "oldpass", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "oldpass")

	status, _ := e.do(client, http.MethodPost, "/account/change-password", map[string]string{
		"oldPassword": "oldpass",
		"newPassword": "newpass",
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	// Old password no longer works.
	client2 := e.newClient()
	status2, _ := e.do(client2, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice@example.com",
		"password": "oldpass",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status2)

	// New password works.
	client3 := e.newClient()
	status3, _ := e.do(client3, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice@example.com",
		"password": "newpass",
	}, nil)
	assert.Equal(t, http.StatusOK, status3)
}

func TestChangePassword_WrongOld(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "oldpass", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "oldpass")

	status, _ := e.do(client, http.MethodPost, "/account/change-password", map[string]string{
		"oldPassword": "wrong",
		"newPassword": "newpass",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestChangePassword_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/account/change-password", map[string]string{
		"oldPassword": "x",
		"newPassword": "y",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestDeleteAccount_RegularUser(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodDelete, "/account", nil, nil)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestDeleteAccount_LastAdmin_Conflict(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	status, _ := e.do(client, http.MethodDelete, "/account", nil, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestDeleteAccount_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodDelete, "/account", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestDeleteAccount_AdminWithSecondAdmin(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin1@example.com", "Admin1", "pass1", true)
	e.createUser("admin2@example.com", "Admin2", "pass2", true)
	client := e.newClient()
	e.login(client, "admin1@example.com", "pass1")

	// Should succeed because there is still another admin.
	status, _ := e.do(client, http.MethodDelete, "/account", nil, nil)
	assert.Equal(t, http.StatusNoContent, status)

	// admin2 can still log in.
	client2 := e.newClient()
	var resp map[string]any
	status2, _ := e.do(client2, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "admin2@example.com",
		"password": "pass2",
	}, &resp)
	require.Equal(t, http.StatusOK, status2)
}
