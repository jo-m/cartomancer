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

const (
	// testBbox is a bounding box covering approximately Switzerland, used across test files.
	testBbox = "5.5,45.5,11.0,48.2"

	// testMaxZoom is the default maximum zoom level used in tests.
	testMaxZoom = 8
)

var testBboxParsed = Bbox{MinLon: 5.5, MinLat: 45.5, MaxLon: 11.0, MaxLat: 48.2}

func TestDbRoundTrip_withBbox(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := context.Background()

	// Verify the table is empty.
	_, err := d.QueryRO().GetLatestReadyMapBuild(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)

	id, err := uuid.NewV7()
	require.NoError(t, err)

	now := time.Now()
	err = d.QueryRW().InsertMapBuild(ctx, db.InsertMapBuildParams{
		Uuid:       id.String(),
		CreatedAt:  now,
		Key:        "20260411.pmtiles",
		Size:       1000,
		Md5sum:     "abc",
		Uploaded:   now,
		Version:    "4.14.5",
		Maxzoom:    8,
		BboxMinLon: testBboxParsed.NullMinLon(),
		BboxMinLat: testBboxParsed.NullMinLat(),
		BboxMaxLon: testBboxParsed.NullMaxLon(),
		BboxMaxLat: testBboxParsed.NullMaxLat(),
	})
	require.NoError(t, err)

	// Not ready yet.
	_, err = d.QueryRO().GetLatestReadyMapBuild(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Look up by key with matching bbox.
	found, err := d.QueryRO().GetMapBuildByKey(ctx, db.GetMapBuildByKeyParams{
		Key:        "20260411.pmtiles",
		Maxzoom:    8,
		BboxMinLon: testBboxParsed.NullMinLon(),
		BboxMinLat: testBboxParsed.NullMinLat(),
		BboxMaxLon: testBboxParsed.NullMaxLon(),
		BboxMaxLat: testBboxParsed.NullMaxLat(),
	})
	require.NoError(t, err)
	require.Equal(t, id.String(), found.Uuid)
	require.Equal(t, int64(0), found.Ready)
	require.True(t, found.BboxMinLon.Valid)
	require.InDelta(t, 5.5, found.BboxMinLon.Float64, 0.001)

	// Mark ready.
	_, err = d.QueryRW().SetMapBuildReady(ctx, id.String())
	require.NoError(t, err)

	ready, err := d.QueryRO().GetLatestReadyMapBuild(ctx)
	require.NoError(t, err)
	require.Equal(t, id.String(), ready.Uuid)
	require.Equal(t, int64(1), ready.Ready)

	// Delete.
	_, err = d.QueryRW().DeleteMapBuild(ctx, id.String())
	require.NoError(t, err)
	_, err = d.QueryRO().GetLatestReadyMapBuild(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDbRoundTrip_nullBbox(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := context.Background()

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
		// All bbox fields left as zero values (null).
	})
	require.NoError(t, err)

	// Look up with null bbox.
	found, err := d.QueryRO().GetMapBuildByKey(ctx, db.GetMapBuildByKeyParams{
		Key:     "20260411.pmtiles",
		Maxzoom: 8,
	})
	require.NoError(t, err)
	require.Equal(t, id.String(), found.Uuid)
	require.False(t, found.BboxMinLon.Valid)
	require.False(t, found.BboxMinLat.Valid)
	require.False(t, found.BboxMaxLon.Valid)
	require.False(t, found.BboxMaxLat.Valid)

	// Should NOT match when looking up with a bbox.
	_, err = d.QueryRO().GetMapBuildByKey(ctx, db.GetMapBuildByKeyParams{
		Key:        "20260411.pmtiles",
		Maxzoom:    8,
		BboxMinLon: testBboxParsed.NullMinLon(),
		BboxMinLat: testBboxParsed.NullMinLat(),
		BboxMaxLon: testBboxParsed.NullMaxLon(),
		BboxMaxLat: testBboxParsed.NullMaxLat(),
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListMapBuilds(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := context.Background()

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
		})
		require.NoError(t, err)
	}

	builds, err := d.QueryRO().ListMapBuilds(ctx)
	require.NoError(t, err)
	require.Len(t, builds, 2)
	require.Equal(t, "20260411.pmtiles", builds[0].Key)
	require.Equal(t, "20260101.pmtiles", builds[1].Key)
}
