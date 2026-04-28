package api

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/blob"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

func TestLoadViewerPoints(t *testing.T) {
	const gpxPath = "../load/testdata/COURSE_436298480.gpx"

	ctx := t.Context()
	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	content, err := os.ReadFile(gpxPath)
	require.NoError(t, err)

	userID, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = d.QueryRW().CreateUser(ctx, db.CreateUserParams{
		Uuid:           userID.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
		Email:          "viewer@example.com",
		Name:           "Viewer",
		PasswordHash:   "x",
		EmailConfirmed: 1,
	})
	require.NoError(t, err)

	b, err := blob.Create(ctx, d.QueryRW(), content, blob.CompressionZstd)
	require.NoError(t, err)

	// Build a tiny set of points and encode it to use as the fast-path polyline.
	encodedPts := track.Points{
		{Lat: 0.0, Lon: 0.0, Elevation: 100},
		{Lat: 0.001, Lon: 0.0, Elevation: 110},
		{Lat: 0.002, Lon: 0.0, Elevation: 120},
		{Lat: 0.003, Lon: 0.0, Elevation: 130},
	}
	encoded, err := track.EncodeVarint(encodedPts)
	require.NoError(t, err)

	trackID, err := uuid.NewV7()
	require.NoError(t, err)
	created, err := d.QueryRW().CreateTrack(ctx, db.CreateTrackParams{
		Uuid:             trackID.String(),
		CreatedAt:        now,
		UpdatedAt:        now,
		UserID:           userID.String(),
		BlobID:           b.ID,
		FileFormat:       int64(track.FileFormatGPX),
		OriginalFilename: "COURSE_436298480.gpx",
		Name:             "test",
		TotalDistanceM:   1000,
		TotalAscentM:     0,
		Public:           1,
	})
	require.NoError(t, err)

	t.Run("decodes precomputed polyline", func(t *testing.T) {
		require.NoError(t, d.QueryRW().SetTrackPreviewPolylines(ctx, db.SetTrackPreviewPolylinesParams{
			Uuid:                created.Uuid,
			PolylineDp5mVarint:  encoded,
			PolylineDp50mVarint: encoded,
		}))

		row, err := d.QueryRO().GetTrackByUUID(ctx, created.Uuid)
		require.NoError(t, err)

		got, err := loadViewerPoints(ctx, d.QueryRO(), row, db.PreviewPolyline5M, track.PointsViewerEpsilonM, 1.0)
		require.NoError(t, err)
		// 1 m thinning preserves all four distinct points.
		require.Len(t, got, 4)
		require.InDelta(t, 0.0, got[0].Lat, 1e-9)
		require.InDelta(t, 0.003, got[len(got)-1].Lat, 1e-9)
	})

	t.Run("falls back to blob when polyline empty", func(t *testing.T) {
		require.NoError(t, d.QueryRW().SetTrackPreviewPolylines(ctx, db.SetTrackPreviewPolylinesParams{
			Uuid:                created.Uuid,
			PolylineDp5mVarint:  []byte{},
			PolylineDp50mVarint: []byte{},
		}))

		row, err := d.QueryRO().GetTrackByUUID(ctx, created.Uuid)
		require.NoError(t, err)

		got, err := loadViewerPoints(ctx, d.QueryRO(), row, db.PreviewPolyline5M, track.PointsViewerEpsilonM, track.PointsViewerMinDistM)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(got), 2)
	})
}
