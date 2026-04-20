package users

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/password"
)

func createTestUser(t *testing.T, d *db.DB, email, name string, confirmed bool) string {
	t.Helper()

	id, err := uuid.NewV7()
	require.NoError(t, err)

	now := time.Now().UTC()
	hash, err := password.Hash("password123")
	require.NoError(t, err)

	var emailConfirmed int64
	if confirmed {
		emailConfirmed = 1
	}

	u, err := d.QueryRW().CreateUser(t.Context(), db.CreateUserParams{
		Uuid:           id.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
		Email:          email,
		Name:           name,
		PasswordHash:   hash,
		Admin:          0,
		EmailConfirmed: emailConfirmed,
	})
	require.NoError(t, err)
	return u.Uuid
}

func createTestVerification(t *testing.T, d *db.DB, userID, email string, expiresAt time.Time) string {
	t.Helper()

	id, err := uuid.NewV7()
	require.NoError(t, err)

	now := time.Now().UTC()
	_, err = d.QueryRW().CreateEmailVerification(t.Context(), db.CreateEmailVerificationParams{
		Uuid:      id.String(),
		CreatedAt: now,
		ExpiresAt: expiresAt,
		UserID:    userID,
		Email:     email,
	})
	require.NoError(t, err)
	return id.String()
}

func TestEmailVerificationCleaner_DeletesExpiredVerifications(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	userID := createTestUser(t, d, "user@example.com", "testuser", true)
	createTestVerification(t, d, userID, "new@example.com", time.Now().Add(-time.Hour))

	cleaner := NewEmailVerificationCleaner(d)
	err := cleaner.Run(context.Background(), emailVerificationCleanerArgs{})
	require.NoError(t, err)

	// Verification should be deleted.
	_, err = d.QueryRO().GetEmailVerificationByUserID(t.Context(), userID)
	require.Error(t, err)

	// User should still exist (they are confirmed).
	_, err = d.QueryRO().GetUser(t.Context(), userID)
	require.NoError(t, err)
}

func TestEmailVerificationCleaner_DeletesUnconfirmedUsersWithoutVerification(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	// Unconfirmed user with expired verification.
	unconfirmedID := createTestUser(t, d, "unconfirmed@example.com", "unconfirmed", false)
	createTestVerification(t, d, unconfirmedID, "unconfirmed@example.com", time.Now().Add(-time.Hour))

	// Unconfirmed user with still-valid verification (should NOT be deleted).
	pendingID := createTestUser(t, d, "pending@example.com", "pending", false)
	createTestVerification(t, d, pendingID, "pending@example.com", time.Now().Add(time.Hour))

	// Confirmed user (should NOT be deleted).
	confirmedID := createTestUser(t, d, "confirmed@example.com", "confirmed", true)

	cleaner := NewEmailVerificationCleaner(d)
	err := cleaner.Run(context.Background(), emailVerificationCleanerArgs{})
	require.NoError(t, err)

	// Unconfirmed user with expired verification should be deleted.
	_, err = d.QueryRO().GetUser(t.Context(), unconfirmedID)
	require.Error(t, err)

	// Pending user should still exist.
	_, err = d.QueryRO().GetUser(t.Context(), pendingID)
	require.NoError(t, err)

	// Confirmed user should still exist.
	_, err = d.QueryRO().GetUser(t.Context(), confirmedID)
	require.NoError(t, err)
}

func TestEmailVerificationCleaner_EmailChangeTimeoutResetsState(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	// Confirmed user who requested an email change but did not confirm it.
	userID := createTestUser(t, d, "original@example.com", "changer", true)
	createTestVerification(t, d, userID, "newemail@example.com", time.Now().Add(-time.Hour))

	cleaner := NewEmailVerificationCleaner(d)
	err := cleaner.Run(context.Background(), emailVerificationCleanerArgs{})
	require.NoError(t, err)

	// User should still exist with their original email.
	u, err := d.QueryRO().GetUser(t.Context(), userID)
	require.NoError(t, err)
	require.Equal(t, "original@example.com", u.Email)

	// Expired verification should be cleaned up.
	_, err = d.QueryRO().GetEmailVerificationByUserID(t.Context(), userID)
	require.Error(t, err)
}
