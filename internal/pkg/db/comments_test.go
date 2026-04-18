package db_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

// createTestComment inserts a comment on the given track by the given user and
// returns the comment UUID.
func createTestComment(t *testing.T, d *db.DB, trackID, userID, body string) string {
	t.Helper()
	id, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()
	err = d.QueryRW().CreateTrackComment(t.Context(), db.CreateTrackCommentParams{
		Uuid:      id.String(),
		TrackID:   trackID,
		UserID:    userID,
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)
	return id.String()
}

func TestListTrackComments_Empty(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")
	trackID := createTestTrack(t, d, alice, 1)

	rows, err := d.QueryRO().ListTrackComments(t.Context(), trackID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestCreateAndListTrackComments(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")
	bob := createTestUser(t, d, "bob@example.com")
	trackID := createTestTrack(t, d, alice, 1)

	c1 := createTestComment(t, d, trackID, alice, "First comment")
	c2 := createTestComment(t, d, trackID, bob, "Second comment")

	rows, err := d.QueryRO().ListTrackComments(t.Context(), trackID)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, c1, rows[0].Uuid)
	require.Equal(t, "First comment", rows[0].Body)
	require.Equal(t, c2, rows[1].Uuid)
	require.Equal(t, "Second comment", rows[1].Body)
	require.Equal(t, int64(0), rows[0].Deleted)
}

func TestGetTrackCommentByUUID(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")
	trackID := createTestTrack(t, d, alice, 1)
	commentID := createTestComment(t, d, trackID, alice, "Hello")

	row, err := d.QueryRO().GetTrackCommentByUUID(t.Context(), commentID)
	require.NoError(t, err)
	require.Equal(t, commentID, row.Uuid)
	require.Equal(t, trackID, row.TrackID)
	require.Equal(t, alice, row.UserID)
	require.Equal(t, "Hello", row.Body)
	require.Equal(t, int64(0), row.Deleted)
}

func TestUpdateTrackCommentBody(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")
	trackID := createTestTrack(t, d, alice, 1)
	commentID := createTestComment(t, d, trackID, alice, "Original")

	n, err := d.QueryRW().UpdateTrackCommentBody(t.Context(), db.UpdateTrackCommentBodyParams{
		Body:      "Updated",
		UpdatedAt: time.Now().UTC(),
		Uuid:      commentID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	row, err := d.QueryRO().GetTrackCommentByUUID(t.Context(), commentID)
	require.NoError(t, err)
	require.Equal(t, "Updated", row.Body)
}

func TestSoftDeleteTrackComment(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")
	trackID := createTestTrack(t, d, alice, 1)
	commentID := createTestComment(t, d, trackID, alice, "To be deleted")

	n, err := d.QueryRW().SoftDeleteTrackComment(t.Context(), db.SoftDeleteTrackCommentParams{
		UpdatedAt: time.Now().UTC(),
		Uuid:      commentID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	row, err := d.QueryRO().GetTrackCommentByUUID(t.Context(), commentID)
	require.NoError(t, err)
	require.Equal(t, "To be deleted", row.Body)
	require.Equal(t, int64(1), row.Deleted)

	// Soft-deleted comments still appear in listing, but body is hidden.
	rows, err := d.QueryRO().ListTrackComments(t.Context(), trackID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0].Deleted)
	require.Equal(t, "", rows[0].Body)
}

func TestCommentsCascadeOnTrackDelete(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")
	trackID := createTestTrack(t, d, alice, 1)
	createTestComment(t, d, trackID, alice, "Comment on track")

	// Delete the track.
	err := d.QueryRW().DeleteTrack(t.Context(), trackID)
	require.NoError(t, err)

	// Comments should be gone.
	rows, err := d.QueryRO().ListTrackComments(t.Context(), trackID)
	require.NoError(t, err)
	require.Empty(t, rows)
}
