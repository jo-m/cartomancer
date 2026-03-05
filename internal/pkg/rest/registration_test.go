package rest_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_Success(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	var resp map[string]any
	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "newuser@example.com",
		"name":     "New User",
		"password": "secret123",
	}, &resp)
	assert.Equal(t, http.StatusCreated, status)
	assert.Equal(t, "check your email", resp["msg"])
}

func TestRegister_DuplicateEmail(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "alice@example.com",
		"name":     "Another Alice",
		"password": "secret123",
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestRegister_Confirm_Login(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	// Register.
	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "bob@example.com",
		"name":     "Bob",
		"password": "secret123",
	}, nil)
	require.Equal(t, http.StatusCreated, status)

	// Login should fail (unconfirmed).
	status, _ = e.do(client, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "bob@example.com",
		"password": "secret123",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)

	// Get token from DB.
	token := e.getVerificationJWT(t, "bob@example.com")

	// Confirm.
	var sessResp map[string]any
	status, _ = e.do(client, http.MethodPost, "/register/confirm", map[string]string{
		"token": token,
	}, &sessResp)
	assert.Equal(t, http.StatusOK, status)
	user, ok := sessResp["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "bob@example.com", user["email"])

	// Login should now succeed (use a fresh client to avoid already-logged-in conflict).
	client2 := e.newClient()
	status, _ = e.do(client2, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "bob@example.com",
		"password": "secret123",
	}, nil)
	assert.Equal(t, http.StatusOK, status)
}

func TestRegister_Confirm_InvalidToken(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/register/confirm", map[string]string{
		"token": "nonexistent",
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestChangeEmail_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	// Request email change.
	status, _ := e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "alice-new@example.com",
		"password": "secret",
	}, nil)
	assert.Equal(t, http.StatusOK, status)

	// Confirm.
	token := e.getVerificationJWT(t, "alice@example.com")
	var resp map[string]any
	status, _ = e.do(client, http.MethodPost, "/account/change-email/confirm", map[string]string{
		"token": token,
	}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "alice-new@example.com", resp["email"])
}

func TestChangeEmail_WrongPassword(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "alice-new@example.com",
		"password": "wrong",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestChangeEmail_EmailTaken(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "bob@example.com",
		"password": "secret",
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestAdminConfirmEmail_Registration(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "secret", true)

	// Register a new user (unconfirmed) first, before creating the admin client.
	client := e.newClient()
	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "pending@example.com",
		"name":     "Pending",
		"password": "secret123",
	}, nil)
	require.Equal(t, http.StatusCreated, status)

	// Get user UUID from DB.
	pendingUser, err := e.d.QueryRO().GetUserByEmail(t.Context(), "pending@example.com")
	require.NoError(t, err)

	// Now log in as admin (reuses the same underlying client, new jar).
	adminClient := e.newClient()
	e.login(adminClient, "admin@example.com", "secret")

	// Admin confirms the email.
	var resp map[string]any
	status, _ = e.do(adminClient, http.MethodPost, "/admin/users/"+pendingUser.Uuid+"/confirm-email", map[string]any{}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "pending@example.com", resp["email"])

	// User can now log in.
	loginClient := e.newClient()
	status, _ = e.do(loginClient, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "pending@example.com",
		"password": "secret123",
	}, nil)
	assert.Equal(t, http.StatusOK, status)
}

func TestAdminConfirmEmail_NoPending(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "secret", true)
	userUUID := e.createUser("user@example.com", "User", "secret", false)
	adminClient := e.newClient()
	e.login(adminClient, "admin@example.com", "secret")

	status, _ := e.do(adminClient, http.MethodPost, "/admin/users/"+userUUID+"/confirm-email", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}
