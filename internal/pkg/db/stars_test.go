package db_test

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

// createTestUser inserts a minimal user and returns its UUID.
// The name is derived from the email local-part so each user has a unique name.
func createTestUser(t *testing.T, d *db.DB, email string) string {
	t.Helper()
	id, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()
	// Use the full UUID without dashes as a guaranteed unique name.
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

// createTestTrack inserts a minimal track and returns its UUID.
func createTestTrack(t *testing.T, d *db.DB, ownerID string, public int64) string {
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
		Uuid:                id.String(),
		CreatedAt:           now,
		UpdatedAt:           now,
		UserID:              ownerID,
		BlobID:              blob.ID,
		FileFormat:          0,
		OriginalFilename:    "test.gpx",
		Name:                "Test Track",
		TotalDistanceM:      1000,
		TotalAscentM:        50,
		Public:              public,
		StartLat:            sql.NullFloat64{Valid: true, Float64: 47.0},
		StartLon:            sql.NullFloat64{Valid: true, Float64: 8.0},
		PolylineDp5mVarint:  []byte{},
		PolylineDp50mVarint: []byte{},
	})
	require.NoError(t, err)
	return tr.Uuid
}

// starTrack inserts a star for the given user and track.
func starTrack(t *testing.T, d *db.DB, userID, trackID string) {
	t.Helper()
	err := d.QueryRW().CreateTrackStar(t.Context(), db.CreateTrackStarParams{
		TrackID:   trackID,
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
	})
	require.NoError(t, err)
}

func TestGetStarredTracks_VisibilityAnonymous(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")
	bob := createTestUser(t, d, "bob@example.com")

	publicTrack := createTestTrack(t, d, bob, 1)
	privateTrack := createTestTrack(t, d, bob, 0)

	starTrack(t, d, alice, publicTrack)
	starTrack(t, d, alice, privateTrack)

	// Anonymous viewer sees only the public star; Starred is always false.
	tracks, err := d.GetStarredTracks(t.Context(), alice, "")
	require.NoError(t, err)
	require.Len(t, tracks, 1)
	require.Equal(t, publicTrack, tracks[0].Uuid)
	require.False(t, tracks[0].Starred)
}

func TestGetStarredTracks_VisibilityOwner(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")

	publicTrack := createTestTrack(t, d, alice, 1)
	privateTrack := createTestTrack(t, d, alice, 0)

	starTrack(t, d, alice, publicTrack)
	starTrack(t, d, alice, privateTrack)

	// Owner viewer sees both stars; Starred is true for all (owner starred them).
	tracks, err := d.GetStarredTracks(t.Context(), alice, alice)
	require.NoError(t, err)
	require.Len(t, tracks, 2)
	require.True(t, tracks[0].Starred)
	require.True(t, tracks[1].Starred)
}

func TestGetStarredTracks_VisibilityOtherUser(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")
	bob := createTestUser(t, d, "bob@example.com")

	publicTrack := createTestTrack(t, d, alice, 1)
	privateTrack := createTestTrack(t, d, alice, 0)

	starTrack(t, d, alice, publicTrack)
	starTrack(t, d, alice, privateTrack)

	// Bob can only see alice's star on the public track; Starred = false since bob hasn't starred it.
	tracks, err := d.GetStarredTracks(t.Context(), alice, bob)
	require.NoError(t, err)
	require.Len(t, tracks, 1)
	require.Equal(t, publicTrack, tracks[0].Uuid)
	require.False(t, tracks[0].Starred)
}

func TestGetStarredTracks_StarredViewerHasStarred(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")
	bob := createTestUser(t, d, "bob@example.com")

	publicTrack := createTestTrack(t, d, alice, 1)

	// Both alice and bob star the track.
	starTrack(t, d, alice, publicTrack)
	starTrack(t, d, bob, publicTrack)

	// Bob views alice's star list; Starred = true since bob also starred this track.
	tracks, err := d.GetStarredTracks(t.Context(), alice, bob)
	require.NoError(t, err)
	require.Len(t, tracks, 1)
	require.True(t, tracks[0].Starred)
}

func TestGetStarredTracks_Empty(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")

	tracks, err := d.GetStarredTracks(t.Context(), alice, "")
	require.NoError(t, err)
	require.Empty(t, tracks)
}

func TestGetTrackByUUIDForViewer_Starred(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d, "alice@example.com")
	bob := createTestUser(t, d, "bob@example.com")

	trackID := createTestTrack(t, d, alice, 1)

	// Bob has not starred: Starred = false.
	tw, err := d.GetTrackByUUIDForViewer(t.Context(), trackID, bob)
	require.NoError(t, err)
	require.False(t, tw.Starred)

	// Bob stars the track: Starred = true.
	starTrack(t, d, bob, trackID)
	tw, err = d.GetTrackByUUIDForViewer(t.Context(), trackID, bob)
	require.NoError(t, err)
	require.True(t, tw.Starred)

	// Anonymous viewer: Starred = false.
	tw, err = d.GetTrackByUUIDForViewer(t.Context(), trackID, "")
	require.NoError(t, err)
	require.False(t, tw.Starred)
}
