package api_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/api"
	"jo-m.ch/go/cartomancer/internal/pkg/app"
	"jo-m.ch/go/cartomancer/internal/pkg/password"
)

func TestRegister_Disabled(t *testing.T) {
	e := newTestEnvWithAppConfig(t, app.AppConfig{
		InstanceName:        "test",
		EmailJWTSecret:      api.TestEmailJWTSecret,
		RegistrationEnabled: false,
	})
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "newuser@example.com",
		"name":     "New-User",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestRegister_Success(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	var resp map[string]any
	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "newuser@example.com",
		"name":     "New-User",
		"password": "secret11",
	}, &resp)
	assert.Equal(t, http.StatusCreated, status)
	assert.Equal(t, "check your email", resp["msg"])
}

func TestRegister_DuplicateEmail(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()

	// Returns 201 even for duplicate email to prevent email enumeration.
	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "alice@example.com",
		"name":     "Another-Alice",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusCreated, status)
}

func TestRegister_Confirm_Login(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	// Register.
	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "bob@example.com",
		"name":     "Bob",
		"password": "secret11",
	}, nil)
	require.Equal(t, http.StatusCreated, status)

	// Login should fail (unconfirmed).
	status, _ = e.do(client, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "bob@example.com",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)

	// Get token from DB.
	token := e.getVerificationJWT(t, "bob@example.com")

	// Confirm (no auto-login, returns message).
	var msgResp map[string]any
	status, _ = e.do(client, http.MethodPost, "/confirm-email", map[string]string{
		"token": token,
	}, &msgResp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "email confirmed", msgResp["msg"])

	// Login should now succeed.
	client2 := e.newClient()
	status, _ = e.do(client2, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "bob@example.com",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusOK, status)
}

func TestRegister_Confirm_InvalidToken(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/confirm-email", map[string]string{
		"token": "nonexistent",
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestChangeEmail_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	// Request email change.
	status, _ := e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "alice-new@example.com",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusOK, status)

	// Confirm via unified endpoint (no auth required).
	token := e.getVerificationJWT(t, "alice@example.com")
	var resp map[string]any
	status, _ = e.do(client, http.MethodPost, "/confirm-email", map[string]string{
		"token": token,
	}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "email confirmed", resp["msg"])

	// Verify email was actually changed.
	client2 := e.newClient()
	status, _ = e.do(client2, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice-new@example.com",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusOK, status)
}

func TestChangeEmail_WrongPassword(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, _ := e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "alice-new@example.com",
		"password": "wrong",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestChangeEmail_EmailTaken(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	e.createUser("bob@example.com", "Bob", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, _ := e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "bob@example.com",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestAdminConfirmEmail_Registration(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "secret11", true)

	// Register a new user (unconfirmed) first, before creating the admin client.
	client := e.newClient()
	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "pending@example.com",
		"name":     "Pending",
		"password": "secret11",
	}, nil)
	require.Equal(t, http.StatusCreated, status)

	// Get user UUID from DB.
	pendingUser, err := e.d.QueryRO().GetUserByEmail(t.Context(), "pending@example.com")
	require.NoError(t, err)

	// Now log in as admin (reuses the same underlying client, new jar).
	adminClient := e.newClient()
	e.login(adminClient, "admin@example.com", "secret11")

	// Admin confirms the email.
	var resp map[string]any
	status, _ = e.do(adminClient, http.MethodPost, "/admin/users/"+pendingUser.Uuid+"/confirm-email", map[string]any{}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "pending@example.com", resp["email"])

	// User can now log in.
	loginClient := e.newClient()
	status, _ = e.do(loginClient, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "pending@example.com",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusOK, status)
}

func TestAdminConfirmEmail_NoPending(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "secret11", true)
	userUUID := e.createUser("user@example.com", "User", "secret11", false)
	adminClient := e.newClient()
	e.login(adminClient, "admin@example.com", "secret11")

	status, _ := e.do(adminClient, http.MethodPost, "/admin/users/"+userUUID+"/confirm-email", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestRegister_DuplicateName(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "other@example.com",
		"name":     "Alice",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestRegister_InvalidName(t *testing.T) {
	tests := []struct {
		desc string
		name string
	}{
		{"too short", "ab"},
		{"too long", strings.Repeat("a", 33)},
		{"consecutive hyphens", "al--ice"},
		{"consecutive underscores", "al__ice"},
		{"mixed consecutive specials", "al-_ice"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			e := newTestEnv(t)
			client := e.newClient()
			status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
				"email":    "user@example.com",
				"name":     tc.name,
				"password": "secret11",
			}, nil)
			assert.Equal(t, http.StatusBadRequest, status)
		})
	}
}

func TestRegister_EmptyFields(t *testing.T) {
	tests := []struct {
		desc string
		body map[string]string
	}{
		{"empty email", map[string]string{"email": "", "name": "Alice", "password": "secret11"}},
		{"empty name", map[string]string{"email": "user@example.com", "name": "", "password": "secret11"}},
		{"empty password", map[string]string{"email": "user@example.com", "name": "Alice", "password": ""}},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			e := newTestEnv(t)
			client := e.newClient()
			status, _ := e.do(client, http.MethodPost, "/register", tc.body, nil)
			assert.Equal(t, http.StatusBadRequest, status)
		})
	}
}

func TestRegister_PasswordTooLong(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "user@example.com",
		"name":     "Alice",
		"password": strings.Repeat("x", password.MaxPasswordLen+1),
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestRegister_ReregisterBeforeConfirm(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "user@example.com",
		"name":     "Alice",
		"password": "secret11",
	}, nil)
	require.Equal(t, http.StatusCreated, status)

	// Second attempt with same email returns 201 to prevent email enumeration.
	// The existing user receives a notification email instead.
	status, _ = e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "user@example.com",
		"name":     "Other-Alice",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusCreated, status)
}

func TestConfirmEmail_ExpiredToken(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "user@example.com",
		"name":     "Alice",
		"password": "secret11",
	}, nil)
	require.Equal(t, http.StatusCreated, status)

	user, err := e.d.QueryRO().GetUserByEmail(t.Context(), "user@example.com")
	require.NoError(t, err)
	ver, err := e.d.QueryRO().GetEmailVerificationByUserID(t.Context(), user.Uuid)
	require.NoError(t, err)
	expiredToken, err := api.SignEmailTokenForTest(ver.Uuid, -time.Hour, e.emailJWTSecret, "test")
	require.NoError(t, err)

	status, _ = e.do(client, http.MethodPost, "/confirm-email", map[string]string{
		"token": expiredToken,
	}, nil)
	assert.Equal(t, http.StatusGone, status)
}

func TestConfirmEmail_TokenReplay(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/register", map[string]string{
		"email":    "user@example.com",
		"name":     "Alice",
		"password": "secret11",
	}, nil)
	require.Equal(t, http.StatusCreated, status)

	token := e.getVerificationJWT(t, "user@example.com")

	status, _ = e.do(client, http.MethodPost, "/confirm-email", map[string]string{"token": token}, nil)
	require.Equal(t, http.StatusOK, status)

	// Second use of the same token must fail because the verification record was deleted.
	status, _ = e.do(client, http.MethodPost, "/confirm-email", map[string]string{"token": token}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestConfirmEmail_EmailChangeTaken(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, _ := e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "wanted@example.com",
		"password": "secret11",
	}, nil)
	require.Equal(t, http.StatusOK, status)

	token := e.getVerificationJWT(t, "alice@example.com")

	// Another user registers with the target email before Alice confirms.
	e.createUser("wanted@example.com", "Bob", "secret11", false)

	status, _ = e.do(client, http.MethodPost, "/confirm-email", map[string]string{"token": token}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestChangeEmail_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "new@example.com",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestChangeEmail_SameEmail(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, _ := e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "alice@example.com",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestAdminConfirmEmail_EmailChange(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	e.createUser("alice@example.com", "Alice", "secret11", false)

	aliceClient := e.newClient()
	e.login(aliceClient, "alice@example.com", "secret11")

	// Alice requests an email change.
	status, _ := e.do(aliceClient, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "alice-new@example.com",
		"password": "secret11",
	}, nil)
	require.Equal(t, http.StatusOK, status)

	aliceUser, err := e.d.QueryRO().GetUserByEmail(t.Context(), "alice@example.com")
	require.NoError(t, err)

	adminClient := e.newClient()
	e.login(adminClient, "admin@example.com", "adminpass")

	// Admin confirms the email-change verification.
	var resp map[string]any
	status, _ = e.do(adminClient, http.MethodPost, "/admin/users/"+aliceUser.Uuid+"/confirm-email", map[string]any{}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "alice-new@example.com", resp["email"])

	// New email works for login.
	loginClient := e.newClient()
	status, _ = e.do(loginClient, http.MethodPost, "/sessions/login", map[string]string{
		"email":    "alice-new@example.com",
		"password": "secret11",
	}, nil)
	assert.Equal(t, http.StatusOK, status)
}

func TestAdminConfirmEmail_AdminTarget(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	e.createUser("admin2@example.com", "Admin2", "adminpass2", true)

	// Admin2 requests an email change to create a pending verification.
	admin2Client := e.newClient()
	e.login(admin2Client, "admin2@example.com", "adminpass2")
	status, _ := e.do(admin2Client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "admin2-new@example.com",
		"password": "adminpass2",
	}, nil)
	require.Equal(t, http.StatusOK, status)

	admin2User, err := e.d.QueryRO().GetUserByEmail(t.Context(), "admin2@example.com")
	require.NoError(t, err)

	adminClient := e.newClient()
	e.login(adminClient, "admin@example.com", "adminpass")

	// Admin confirm email for an admin target must be forbidden.
	status, _ = e.do(adminClient, http.MethodPost, "/admin/users/"+admin2User.Uuid+"/confirm-email", map[string]any{}, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestAdminConfirmEmail_EmailTaken(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	e.createUser("alice@example.com", "Alice", "secret11", false)

	aliceClient := e.newClient()
	e.login(aliceClient, "alice@example.com", "secret11")

	// Alice requests an email change.
	status, _ := e.do(aliceClient, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "wanted@example.com",
		"password": "secret11",
	}, nil)
	require.Equal(t, http.StatusOK, status)

	// Another user takes the target email before admin confirms.
	e.createUser("wanted@example.com", "Bob", "secret11", false)

	aliceUser, err := e.d.QueryRO().GetUserByEmail(t.Context(), "alice@example.com")
	require.NoError(t, err)

	adminClient := e.newClient()
	e.login(adminClient, "admin@example.com", "adminpass")

	status, _ = e.do(adminClient, http.MethodPost, "/admin/users/"+aliceUser.Uuid+"/confirm-email", map[string]any{}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestAdminConfirmEmail_NonAdmin(t *testing.T) {
	e := newTestEnv(t)
	userUUID := e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, _ := e.do(client, http.MethodPost, "/admin/users/"+userUUID+"/confirm-email", map[string]any{}, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestChangeEmail_SecondRequestSupersedes(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	// First email change request.
	status, _ := e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "new1@example.com",
		"password": "secret11",
	}, nil)
	require.Equal(t, http.StatusOK, status)

	// Capture the token for the first request before it is overwritten.
	firstToken := e.getVerificationJWT(t, "alice@example.com")

	// Second email change request supersedes the first.
	status, _ = e.do(client, http.MethodPost, "/account/change-email", map[string]string{
		"newEmail": "new2@example.com",
		"password": "secret11",
	}, nil)
	require.Equal(t, http.StatusOK, status)

	// The first verification was deleted; confirming with its token must fail.
	status, _ = e.do(client, http.MethodPost, "/confirm-email", map[string]string{"token": firstToken}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}
