package roadclosures_test

import (
	"errors"
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures"
)

func TestInsert_NilGeometryRejected(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	ctx := t.Context()
	now := time.Now()

	err := d.WithTx(ctx, func(tx *db.Queries) error {
		return roadclosures.Insert(ctx, tx, roadclosures.ClosureInsert{
			SourceID:    "no-geom",
			InsertedBy:  "test",
			Type:        roadclosures.ClosedWay,
			Title:       "no geometry",
			Attribution: attribute.Attribution{Author: "x", Source: "y"},
		}, now)
	})
	require.True(t, errors.Is(err, roadclosures.ErrNilGeometry))

	// No row should have been written for the failing call.
	count, err := d.QueryRO().CountRoadClosures(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestInsert_WithGeometryWritesRowAndCells(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	ctx := t.Context()
	now := time.Now()

	geom := geojson.NewGeometry(orb.LineString{{8.5, 47.3}, {8.52, 47.32}})

	err := d.WithTx(ctx, func(tx *db.Queries) error {
		return roadclosures.Insert(ctx, tx, roadclosures.ClosureInsert{
			SourceID:    "with-geom",
			InsertedBy:  "test",
			Type:        roadclosures.ClosedWay,
			Title:       "has geometry",
			Geometry:    geom,
			Attribution: attribute.Attribution{Author: "x", Source: "y"},
		}, now)
	})
	require.NoError(t, err)

	rows, err := d.QueryRO().CountRoadClosures(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)

	cells, err := d.QueryRO().CountRoadClosureCellsRes7(ctx)
	require.NoError(t, err)
	require.Greater(t, cells, int64(0), "expected at least one cell")
}
