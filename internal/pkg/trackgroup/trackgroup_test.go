package trackgroup

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

// gpxLine generates a minimal GPX file with points along a line between two coordinates.
func gpxLine(lat0, lon0, lat1, lon1 float64, n int) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1">`)
	b.WriteString(`<trk><trkseg>`)
	t := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := range n {
		f := float64(i) / float64(n-1)
		lat := lat0 + f*(lat1-lat0)
		lon := lon0 + f*(lon1-lon0)
		ts := t.Add(time.Duration(i) * time.Second)
		fmt.Fprintf(&b, `<trkpt lat="%.6f" lon="%.6f"><ele>100</ele><time>%s</time></trkpt>`,
			lat, lon, ts.Format(time.RFC3339))
	}
	b.WriteString(`</trkseg></trk></gpx>`)
	return []byte(b.String())
}

func createTestUser(t *testing.T, d *db.DB) string {
	t.Helper()
	id, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()
	name := strings.ReplaceAll(id.String(), "-", "")
	u, err := d.QueryRW().CreateUser(t.Context(), db.CreateUserParams{
		Uuid:           id.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
		Email:          name + "@test.example",
		Name:           name,
		PasswordHash:   "hash",
		Admin:          0,
		EmailConfirmed: 1,
	})
	require.NoError(t, err)
	return u.Uuid
}

// createTestTrack inserts a track with real GPX content so it can be loaded and converted to cells.
// The track is marked as editing-completed so it qualifies for grouping.
func createTestTrack(t *testing.T, d *db.DB, ownerID string, gpx []byte, distM float64) string {
	t.Helper()
	id, err := uuid.NewV7()
	require.NoError(t, err)

	blob, err := d.QueryRW().CreateBlob(t.Context(), db.CreateBlobParams{
		Compression: 0,
		Content:     gpx,
		HashType:    0,
		Hash:        []byte(id.String()),
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
		TotalDistanceM:      distM,
		TotalAscentM:        0,
		Public:              0,
		StartLat:            sql.NullFloat64{Valid: true, Float64: 47.0},
		StartLon:            sql.NullFloat64{Valid: true, Float64: 8.0},
		PolylineDp5mVarint:  []byte{},
		PolylineDp50mVarint: []byte{},
	})
	require.NoError(t, err)

	err = d.CompleteEditing(t.Context(), ownerID, []string{tr.Uuid})
	require.NoError(t, err)

	return tr.Uuid
}

func TestGroupUser_NoTracks(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	userID := createTestUser(t, d)
	err := GroupUser(t.Context(), d, userID)
	require.NoError(t, err)

	rows, err := d.QueryRO().ListTrackGroupsByUser(t.Context(), userID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestGroupUser_SingleTrack(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	userID := createTestUser(t, d)
	gpx := gpxLine(52.50, 13.00, 52.50, 14.00, 500)
	createTestTrack(t, d, userID, gpx, 5000)

	err := GroupUser(t.Context(), d, userID)
	require.NoError(t, err)

	rows, err := d.QueryRO().ListTrackGroupsByUser(t.Context(), userID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestGroupUser_IdenticalTracks(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	userID := createTestUser(t, d)
	gpx := gpxLine(52.50, 13.00, 52.50, 14.00, 500)
	id1 := createTestTrack(t, d, userID, gpx, 5000)
	id2 := createTestTrack(t, d, userID, gpx, 5000)

	err := GroupUser(t.Context(), d, userID)
	require.NoError(t, err)

	rows, err := d.QueryRO().ListTrackGroupsByUser(t.Context(), userID)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, rows[0].Uuid, rows[1].Uuid)
	trackIDs := []string{rows[0].TrackID, rows[1].TrackID}
	require.ElementsMatch(t, []string{id1, id2}, trackIDs)
}

func TestGroupUser_DisjointTracks(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	userID := createTestUser(t, d)
	gpxA := gpxLine(52.50, 13.00, 52.50, 14.00, 500)
	gpxB := gpxLine(48.10, 11.00, 48.10, 12.00, 500)
	createTestTrack(t, d, userID, gpxA, 5000)
	createTestTrack(t, d, userID, gpxB, 5000)

	err := GroupUser(t.Context(), d, userID)
	require.NoError(t, err)

	rows, err := d.QueryRO().ListTrackGroupsByUser(t.Context(), userID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestGroupUser_ExcludesLongTracks(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	userID := createTestUser(t, d)
	gpx := gpxLine(52.50, 13.00, 52.50, 14.00, 500)
	// One track is under the limit, the other exceeds it.
	createTestTrack(t, d, userID, gpx, 5000)
	createTestTrack(t, d, userID, gpx, maxTrackDistanceM+1)

	err := GroupUser(t.Context(), d, userID)
	require.NoError(t, err)

	// Only one track qualifies, so no group can be formed.
	rows, err := d.QueryRO().ListTrackGroupsByUser(t.Context(), userID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestGroupUser_RegroupsAfterNewTrack(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	userID := createTestUser(t, d)
	gpx := gpxLine(52.50, 13.00, 52.50, 14.00, 500)
	createTestTrack(t, d, userID, gpx, 5000)
	createTestTrack(t, d, userID, gpx, 5000)

	err := GroupUser(t.Context(), d, userID)
	require.NoError(t, err)
	rows1, err := d.QueryRO().ListTrackGroupsByUser(t.Context(), userID)
	require.NoError(t, err)
	require.Len(t, rows1, 2)

	// Add a third track; grouping should produce new group UUIDs.
	createTestTrack(t, d, userID, gpx, 5000)
	err = GroupUser(t.Context(), d, userID)
	require.NoError(t, err)
	rows2, err := d.QueryRO().ListTrackGroupsByUser(t.Context(), userID)
	require.NoError(t, err)
	require.Len(t, rows2, 3)
	require.NotEqual(t, rows1[0].Uuid, rows2[0].Uuid)
}

func TestGroupUser_DoesNotAffectOtherUsers(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	alice := createTestUser(t, d)
	bob := createTestUser(t, d)

	gpx := gpxLine(52.50, 13.00, 52.50, 14.00, 500)
	createTestTrack(t, d, alice, gpx, 5000)
	createTestTrack(t, d, alice, gpx, 5000)
	createTestTrack(t, d, bob, gpx, 5000)
	createTestTrack(t, d, bob, gpx, 5000)

	// Group both users.
	err := GroupUser(t.Context(), d, alice)
	require.NoError(t, err)
	err = GroupUser(t.Context(), d, bob)
	require.NoError(t, err)

	aliceRows, err := d.QueryRO().ListTrackGroupsByUser(t.Context(), alice)
	require.NoError(t, err)
	require.Len(t, aliceRows, 2)

	bobRows, err := d.QueryRO().ListTrackGroupsByUser(t.Context(), bob)
	require.NoError(t, err)
	require.Len(t, bobRows, 2)

	// Re-grouping alice should not touch bob's groups.
	err = GroupUser(t.Context(), d, alice)
	require.NoError(t, err)
	bobRows2, err := d.QueryRO().ListTrackGroupsByUser(t.Context(), bob)
	require.NoError(t, err)
	require.Equal(t, bobRows[0].Uuid, bobRows2[0].Uuid)
}
