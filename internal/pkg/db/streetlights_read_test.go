package db_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

// seedStreetlight inserts one streetlight row with the given cell index,
// bypassing the streetlights package so tests can use arbitrary cell values.
func seedStreetlight(t *testing.T, d *db.DB, cell int64, geometry, attribution, href string) {
	t.Helper()
	ctx := t.Context()
	err := d.WithTx(ctx, func(q *db.Queries) error {
		attrID, err := q.UpsertStreetlightAttribution(ctx, db.UpsertStreetlightAttributionParams{
			Attribution:     attribution,
			AttributionHref: href,
		})
		if err != nil {
			return err
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		return q.InsertStreetlight(ctx, db.InsertStreetlightParams{
			Uuid:          id.String(),
			SourceID:      "test",
			InsertedBy:    "test",
			CreatedAt:     time.Now(),
			AttributionID: attrID,
			Cell:          cell,
			Geometry:      geometry,
		})
	})
	require.NoError(t, err)
}

func TestGetStreetlightsByCells(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })
	ctx := t.Context()

	out, err := d.GetStreetlightsByCells(ctx, nil)
	require.NoError(t, err)
	require.Nil(t, out)

	seedStreetlight(t, d, 10, `{"type":"Point","coordinates":[8.5,47.3]}`, "Source A", "https://a.example")
	seedStreetlight(t, d, 20, `{"type":"Point","coordinates":[8.6,47.3]}`, "Source B", "https://b.example")
	seedStreetlight(t, d, 30, `{"type":"Point","coordinates":[8.7,47.3]}`, "Source B", "https://b.example")

	// A subset query must return only matching rows, joined with the
	// attribution of their source.
	out, err = d.GetStreetlightsByCells(ctx, []int64{20, 30, 999})
	require.NoError(t, err)
	require.Len(t, out, 2)
	for _, row := range out {
		require.Contains(t, []int64{20, 30}, row.Cell)
		require.Equal(t, "Source B", row.Attribution)
		require.Equal(t, "https://b.example", row.AttributionHref)
	}

	// Duplicate input cells must not duplicate rows.
	out, err = d.GetStreetlightsByCells(ctx, []int64{20, 20, 20})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, int64(20), out[0].Cell)

	// An unmatched cell list must return nothing.
	out, err = d.GetStreetlightsByCells(ctx, []int64{999})
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestGetStreetlightsByCells_Chunking(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })
	ctx := t.Context()

	// The lamp's cell must land after the first chunk of the query cell list
	// so that finding it proves the multi-statement loop works.
	const lampCell = 1000000
	seedStreetlight(t, d, lampCell, `{"type":"Point","coordinates":[8.5,47.3]}`, "Source", "https://example.com")

	cells := make([]int64, 0, 1201)
	for i := int64(0); i < 1200; i++ {
		cells = append(cells, i)
	}
	cells = append(cells, lampCell)

	out, err := d.GetStreetlightsByCells(ctx, cells)
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, int64(lampCell), out[0].Cell)
}

func TestGetStreetlightDataFingerprint(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })
	ctx := t.Context()

	count, latestMs, err := d.GetStreetlightDataFingerprint(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
	require.Zero(t, latestMs)

	seedStreetlight(t, d, 10, `{"type":"Point","coordinates":[8.5,47.3]}`, "Source", "https://example.com")
	count, latestMs, err = d.GetStreetlightDataFingerprint(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	require.Positive(t, latestMs)

	seedStreetlight(t, d, 20, `{"type":"Point","coordinates":[8.6,47.3]}`, "Source", "https://example.com")
	count, latestMs2, err := d.GetStreetlightDataFingerprint(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.GreaterOrEqual(t, latestMs2, latestMs)
}
