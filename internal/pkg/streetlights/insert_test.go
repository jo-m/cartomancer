package streetlights_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/streetlights"
)

func TestInsert_NilGeometryRejected(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	ctx := t.Context()

	err := d.WithTx(ctx, func(tx *db.Queries) error {
		return streetlights.Insert(ctx, tx, streetlights.StreetlightInsert{
			SourceID:    "no-geom",
			InsertedBy:  "test",
			Attribution: attribute.Attribution{Author: "x", Source: "y"},
		}, time.Now())
	})
	require.True(t, errors.Is(err, streetlights.ErrNilGeometry))

	// No row should have been written for the failing call.
	count, err := d.QueryRO().CountStreetlights(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestInsert_NonPointGeometryRejected(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	ctx := t.Context()
	geom := geojson.NewGeometry(orb.LineString{{8.5, 47.3}, {8.52, 47.32}})

	err := d.WithTx(ctx, func(tx *db.Queries) error {
		return streetlights.Insert(ctx, tx, streetlights.StreetlightInsert{
			SourceID:    "line",
			InsertedBy:  "test",
			Geometry:    geom,
			Attribution: attribute.Attribution{Author: "x", Source: "y"},
		}, time.Now())
	})
	require.True(t, errors.Is(err, streetlights.ErrNonPointGeometry))

	count, err := d.QueryRO().CountStreetlights(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestInsert_InvalidPropertiesRejected(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	ctx := t.Context()
	geom := geojson.NewGeometry(orb.Point{8.5, 47.3})

	for _, props := range []json.RawMessage{json.RawMessage("[1,2]"), json.RawMessage("not json")} {
		err := d.WithTx(ctx, func(tx *db.Queries) error {
			return streetlights.Insert(ctx, tx, streetlights.StreetlightInsert{
				SourceID:    "bad-props",
				InsertedBy:  "test",
				Properties:  props,
				Geometry:    geom,
				Attribution: attribute.Attribution{Author: "x", Source: "y"},
			}, time.Now())
		})
		require.ErrorContains(t, err, "not a valid JSON object", "props=%q", props)
	}

	count, err := d.QueryRO().CountStreetlights(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestInsert_PointWritesRowAndCell(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	ctx := t.Context()
	geom := geojson.NewGeometry(orb.Point{8.5, 47.3})

	err := d.WithTx(ctx, func(tx *db.Queries) error {
		return streetlights.Insert(ctx, tx, streetlights.StreetlightInsert{
			SourceID:        "ktzh-1",
			InsertedBy:      "test",
			ContentProvider: streetlights.NullString("Tiefbauamt"),
			Properties:      json.RawMessage(`{"nummer":"504"}`),
			Geometry:        geom,
			Attribution:     attribute.Attribution{Author: "x", Source: "y"},
		}, time.Now())
	})
	require.NoError(t, err)

	rows, err := d.QueryRO().CountStreetlights(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)

	cells, err := d.QueryRO().CountStreetlightCellsRes9(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), cells, "a point should map to exactly one cell")

	// Properties and provider must round-trip unchanged.
	var props string
	var provider string
	err = d.RO().QueryRowContext(ctx, "SELECT properties, content_provider FROM streetlights").Scan(&props, &provider)
	require.NoError(t, err)
	require.JSONEq(t, `{"nummer":"504"}`, props)
	require.Equal(t, "Tiefbauamt", provider)
}

func TestInsert_MultiPointWritesOneCellPerPoint(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	ctx := t.Context()
	geom := geojson.NewGeometry(orb.MultiPoint{{8.5, 47.3}, {9.0, 46.5}})

	err := d.WithTx(ctx, func(tx *db.Queries) error {
		return streetlights.Insert(ctx, tx, streetlights.StreetlightInsert{
			SourceID:    "multi",
			InsertedBy:  "test",
			Geometry:    geom,
			Attribution: attribute.Attribution{Author: "x", Source: "y"},
		}, time.Now())
	})
	require.NoError(t, err)

	cells, err := d.QueryRO().CountStreetlightCellsRes9(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), cells)
}

func TestInsert_EmptyPropertiesStoredAsEmptyObject(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	ctx := t.Context()
	geom := geojson.NewGeometry(orb.Point{8.5, 47.3})

	props := []json.RawMessage{nil, json.RawMessage("null"), json.RawMessage("  ")}
	srcIDs := []string{"a", "b", "c"}
	for i, p := range props {
		err := d.WithTx(ctx, func(tx *db.Queries) error {
			return streetlights.Insert(ctx, tx, streetlights.StreetlightInsert{
				SourceID:    srcIDs[i],
				InsertedBy:  "test",
				Properties:  p,
				Geometry:    geom,
				Attribution: attribute.Attribution{Author: "x", Source: "y"},
			}, time.Now())
		})
		require.NoError(t, err)
	}

	rows, err := d.RO().QueryContext(ctx, "SELECT properties FROM streetlights ORDER BY source_id")
	require.NoError(t, err)
	defer rows.Close()
	var n int
	for rows.Next() {
		var got string
		require.NoError(t, rows.Scan(&got))
		require.Equal(t, "{}", got)
		n++
	}
	require.NoError(t, rows.Err())
	require.Equal(t, 3, n)
}

func TestNullString(t *testing.T) {
	require.False(t, streetlights.NullString("").Valid)
	require.True(t, streetlights.NullString("x").Valid)
	require.Equal(t, "x", streetlights.NullString("x").String)
}
