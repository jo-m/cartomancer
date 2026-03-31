package app

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

func createTestUser(t *testing.T, d *db.DB, email string) string {
	t.Helper()
	id, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()
	name := strings.ReplaceAll(id.String(), "-", "")
	u, err := d.QueryRW().CreateUser(t.Context(), db.CreateUserParams{
		Uuid:           id.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
		Email:          email,
		Name:           name,
		PasswordHash:   "hash",
		Admin:          0,
		EmailConfirmed: 1,
	})
	require.NoError(t, err)
	return u.Uuid
}

func createTestTrack(t *testing.T, d *db.DB, ownerID string) string {
	t.Helper()
	id, err := uuid.NewV7()
	require.NoError(t, err)
	blob, err := d.QueryRW().CreateBlob(t.Context(), db.CreateBlobParams{
		Compression: 0,
		Content:     []byte("dummy"),
		HashType:    0,
		Hash:        []byte("hash"),
	})
	require.NoError(t, err)
	now := time.Now().UTC()
	tr, err := d.QueryRW().CreateTrack(t.Context(), db.CreateTrackParams{
		Uuid:             id.String(),
		CreatedAt:        now,
		UpdatedAt:        now,
		UserID:           ownerID,
		BlobID:           blob.ID,
		FileFormat:       0,
		OriginalFilename: "test.gpx",
		Name:             "Test Track",
		TotalDistanceM:   1000,
		TotalAscentM:     50,
		Public:           0,
	})
	require.NoError(t, err)
	return tr.Uuid
}

func TestInstallDemoTriggers_BlocksUserInsert(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })
	ctx := t.Context()

	err := InstallDemoTriggers(ctx, d.RW())
	require.NoError(t, err)

	id, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = d.QueryRW().CreateUser(ctx, db.CreateUserParams{
		Uuid:         id.String(),
		CreatedAt:    now,
		UpdatedAt:    now,
		Email:        "new@example.com",
		Name:         "New User",
		PasswordHash: "hash",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "demo mode")
}

func TestInstallDemoTriggers_BlocksUserDelete(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })
	ctx := t.Context()

	userID := createTestUser(t, d, "user@example.com")

	err := InstallDemoTriggers(ctx, d.RW())
	require.NoError(t, err)

	_, err = d.QueryRW().DeleteUser(ctx, userID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "demo mode")
}

func TestInstallDemoTriggers_BlocksUserUpdate(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })
	ctx := t.Context()

	userID := createTestUser(t, d, "user@example.com")

	err := InstallDemoTriggers(ctx, d.RW())
	require.NoError(t, err)

	// Updating the email (a protected field) must fail.
	_, err = d.QueryRW().UpdateUserEmail(ctx, db.UpdateUserEmailParams{
		Email:     "changed@example.com",
		UpdatedAt: time.Now().UTC(),
		Uuid:      userID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "demo mode")
}

func TestInstallDemoTriggers_AllowsLoginTimestamps(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })
	ctx := t.Context()

	userID := createTestUser(t, d, "user@example.com")

	err := InstallDemoTriggers(ctx, d.RW())
	require.NoError(t, err)

	// Updating last_login_at must succeed.
	now := sql.NullTime{Valid: true, Time: time.Now().UTC()}
	err = db.EnsureOneRowChanged(d.QueryRW().UpdateUserLastLogin(ctx, db.UpdateUserLastLoginParams{
		LastLoginAt:  now,
		LastActiveAt: now,
		Uuid:         userID,
	}))
	require.NoError(t, err)

	// Updating last_active_at must succeed.
	err = db.EnsureOneRowChanged(d.QueryRW().UpdateUserLastActive(ctx, db.UpdateUserLastActiveParams{
		LastActiveAt: now,
		Uuid:         userID,
	}))
	require.NoError(t, err)
}

func TestInstallDemoTriggers_AllowsSessions(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })
	ctx := t.Context()

	userID := createTestUser(t, d, "user@example.com")

	err := InstallDemoTriggers(ctx, d.RW())
	require.NoError(t, err)

	// Creating a session must succeed.
	sessID, err := uuid.NewV7()
	require.NoError(t, err)
	_, err = d.QueryRW().CreateSession(ctx, db.CreateSessionParams{
		Uuid:         sessID.String(),
		CreatedAt:    time.Now().UTC(),
		LastActiveAt: time.Now().UTC(),
		UserID:       sql.NullString{Valid: true, String: userID},
	})
	require.NoError(t, err)

	// Deleting a session must succeed.
	_, err = d.QueryRW().DeleteSession(ctx, sessID.String())
	require.NoError(t, err)
}

func TestInstallDemoTriggers_BlocksEmailVerification(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })
	ctx := t.Context()

	userID := createTestUser(t, d, "user@example.com")

	err := InstallDemoTriggers(ctx, d.RW())
	require.NoError(t, err)

	id, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = d.QueryRW().CreateEmailVerification(ctx, db.CreateEmailVerificationParams{
		Uuid:      id.String(),
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
		UserID:    userID,
		Email:     "test@example.com",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "demo mode")
}

func TestDemoTrackPurger_DeletesAllTracks(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })
	ctx := t.Context()

	userID := createTestUser(t, d, "user@example.com")
	createTestTrack(t, d, userID)
	createTestTrack(t, d, userID)

	count, err := d.QueryRO().CountTracksByUser(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	purger := NewDemoTrackPurger(d)
	err = purger.Run(ctx, DemoTrackPurgeArgs{})
	require.NoError(t, err)

	count, err = d.QueryRO().CountTracksByUser(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

func TestDemoTrackPurger_NoTracksIsNoOp(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	purger := NewDemoTrackPurger(d)
	err := purger.Run(t.Context(), DemoTrackPurgeArgs{})
	require.NoError(t, err)
}
