package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/password"
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
