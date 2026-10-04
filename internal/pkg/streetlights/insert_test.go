package streetlights_test

import (
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/require"
	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/streetlights"
)

// testAttribution is the data source credit used across the tests.
var testAttribution = attribute.Attribution{Author: "Tiefbauamt", Source: "https://example.com/data"}

// pointInsert builds a minimal streetlight insert for a single point.
func pointInsert(sourceID string, ll orb.Point) streetlights.StreetlightInsert {
	return streetlights.StreetlightInsert{
		SourceID:    sourceID,
		InsertedBy:  "test",
		Geometry:    geojson.NewGeometry(ll),
		Attribution: testAttribution,
	}
}

// count returns the number of rows in a table.
func count(t *testing.T, d *db.DB, table string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, d.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM "+table).Scan(&n))
	return n
}

func TestInsert_PointWritesRowCellAndSource(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })
	ctx := t.Context()

	ll := orb.Point{8.5, 47.3}
	err := d.WithTx(ctx, func(tx *db.Queries) error {
		return streetlights.Insert(ctx, tx, pointInsert("ktzh-1", ll), time.Now())
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), count(t, d, "streetlights"))

	// The stored cell must be the res-9 cell of the lamp position, and the
	// attribution must resolve through its lookup table.
	want, err := h3.LatLngToCell(h3.LatLng{Lat: ll.Lat(), Lng: ll.Lon()}, streetlights.CellResolution)
	require.NoError(t, err)

	var cell int64
	var geom string
	var attr, href string
	err = d.RO().QueryRowContext(ctx, `
		SELECT s.cell, s.geometry, a.attribution, a.attribution_href
		FROM streetlights s
		JOIN streetlight_attributions a ON a.id = s.attribution_id`).
		Scan(&cell, &geom, &attr, &href)
	require.NoError(t, err)
	require.Equal(t, int64(want), cell)
	require.JSONEq(t, `{"type":"Point","coordinates":[8.5,47.3]}`, geom)
	require.Equal(t, "Tiefbauamt", attr)
	require.Equal(t, "https://example.com/data", href)
}

func TestInsert_AttributionDeduped(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })
	ctx := t.Context()

	insert := func(s streetlights.StreetlightInsert) {
		t.Helper()
		err := d.WithTx(ctx, func(tx *db.Queries) error {
			return streetlights.Insert(ctx, tx, s, time.Now())
		})
		require.NoError(t, err)
	}

	insert(pointInsert("a", orb.Point{8.5, 47.3}))
	insert(pointInsert("b", orb.Point{8.6, 47.3}))

	other := pointInsert("c", orb.Point{8.7, 47.3})
	other.Attribution = attribute.Attribution{Author: "ewz", Source: "https://example.com/other"}
	insert(other)

	require.Equal(t, int64(3), count(t, d, "streetlights"))
	require.Equal(t, int64(2), count(t, d, "streetlight_attributions"))
}

func TestInsert_NilGeometryRejected(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })
	ctx := t.Context()

	s := pointInsert("no-geom", orb.Point{})
	s.Geometry = nil
	err := d.WithTx(ctx, func(tx *db.Queries) error {
		return streetlights.Insert(ctx, tx, s, time.Now())
	})
	require.ErrorIs(t, err, streetlights.ErrNilGeometry)

	// The geometry is validated before any writes, so nothing was written.
	require.Zero(t, count(t, d, "streetlights"))
	require.Zero(t, count(t, d, "streetlight_attributions"))
}

func TestInsert_NonPointGeometryRejected(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })
	ctx := t.Context()

	geoms := []orb.Geometry{
		orb.LineString{{8.5, 47.3}, {8.52, 47.32}},
		orb.MultiPoint{{8.5, 47.3}, {8.6, 47.4}},
	}
	for _, geom := range geoms {
		s := pointInsert("non-point", orb.Point{})
		s.Geometry = geojson.NewGeometry(geom)
		err := d.WithTx(ctx, func(tx *db.Queries) error {
			return streetlights.Insert(ctx, tx, s, time.Now())
		})
		require.ErrorIs(t, err, streetlights.ErrNonPointGeometry, "geometry=%T", geom)
	}

	// The geometry is validated before any writes, so nothing was written.
	require.Zero(t, count(t, d, "streetlights"))
	require.Zero(t, count(t, d, "streetlight_attributions"))
}

// TestInsert_RejectionDoesNotAbortTx mirrors the downloader loop, which
// continues after a rejected feature within the same transaction.
func TestInsert_RejectionDoesNotAbortTx(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })
	ctx := t.Context()

	err := d.WithTx(ctx, func(tx *db.Queries) error {
		bad := pointInsert("bad", orb.Point{})
		bad.Geometry = geojson.NewGeometry(orb.MultiPoint{{8.5, 47.3}})
		if err := streetlights.Insert(ctx, tx, bad, time.Now()); err == nil {
			t.Fatal("expected non-point geometry to be rejected")
		}
		return streetlights.Insert(ctx, tx, pointInsert("good", orb.Point{8.5, 47.3}), time.Now())
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), count(t, d, "streetlights"))
}
