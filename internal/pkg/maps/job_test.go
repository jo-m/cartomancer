package maps

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

func TestExtractAndRecord_dbRoundTrip(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := context.Background()

	// Verify the table is empty.
	_, err := d.QueryRO().GetLatestReadyMapBuild(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Insert a build record manually.
	id, err := uuid.NewV7()
	require.NoError(t, err)

	now := time.Now()
	err = d.QueryRW().InsertMapBuild(ctx, db.InsertMapBuildParams{
		Uuid:      id.String(),
		CreatedAt: now,
		Key:       "20260411.pmtiles",
		Size:      1000,
		Md5sum:    "abc",
		Uploaded:  now,
		Version:   "4.14.5",
		Maxzoom:   8,
		Bbox:      testBbox,
	})
	require.NoError(t, err)

	// Not ready yet.
	_, err = d.QueryRO().GetLatestReadyMapBuild(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Look up by key.
	found, err := d.QueryRO().GetMapBuildByKey(ctx, db.GetMapBuildByKeyParams{
		Key:     "20260411.pmtiles",
		Maxzoom: 8,
		Bbox:    testBbox,
	})
	require.NoError(t, err)
	require.Equal(t, id.String(), found.Uuid)
	require.Equal(t, int64(0), found.Ready)

	// Mark ready.
	_, err = d.QueryRW().SetMapBuildReady(ctx, id.String())
	require.NoError(t, err)

	// Now it should be found as ready.
	ready, err := d.QueryRO().GetLatestReadyMapBuild(ctx)
	require.NoError(t, err)
	require.Equal(t, id.String(), ready.Uuid)
	require.Equal(t, int64(1), ready.Ready)
	require.Equal(t, "20260411.pmtiles", ready.Key)

	// Delete.
	_, err = d.QueryRW().DeleteMapBuild(ctx, id.String())
	require.NoError(t, err)

	// Should be gone.
	_, err = d.QueryRO().GetLatestReadyMapBuild(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListMapBuilds(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := context.Background()

	// Insert two builds.
	for i, key := range []string{"20260101.pmtiles", "20260411.pmtiles"} {
		id, err := uuid.NewV7()
		require.NoError(t, err)

		now := time.Now().Add(time.Duration(i) * time.Hour)
		err = d.QueryRW().InsertMapBuild(ctx, db.InsertMapBuildParams{
			Uuid:      id.String(),
			CreatedAt: now,
			Key:       key,
			Size:      int64(1000 * (i + 1)),
			Md5sum:    "md5",
			Uploaded:  now,
			Version:   "1.0.0",
			Maxzoom:   8,
			Bbox:      testBbox,
		})
		require.NoError(t, err)
	}

	builds, err := d.QueryRO().ListMapBuilds(ctx)
	require.NoError(t, err)
	require.Len(t, builds, 2)
	// Ordered by uploaded DESC.
	require.Equal(t, "20260411.pmtiles", builds[0].Key)
	require.Equal(t, "20260101.pmtiles", builds[1].Key)
}

func TestDownloaderRun_disabled(t *testing.T) {
	d := db.GetTestDB(t)
	dl := NewDownloader(d, MapsConfig{}, t.TempDir())
	err := dl.Run(context.Background(), DownloaderArgs{})
	require.NoError(t, err)
}
