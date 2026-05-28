//go:build online

package tg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures/tg"
)

// TestOnlineFetch hits the live ThurGIS identify endpoint, verifies that at
// least one feature comes back, and asserts that reprojected geometries are
// well-formed WGS84 coordinates inside Switzerland. This catches both a
// silent change of the upstream srs and a regression in the LV95->WGS84
// pipeline.
func TestOnlineFetch(t *testing.T) {
	ctx := context.Background()

	resp, err := tg.Fetch(ctx)
	require.NoError(t, err)
	require.True(t, resp.Success, "expected success=true in response envelope")
	require.NotEmpty(t, resp.Results, "expected at least one feature")

	// Switzerland's WGS84 bounding box (loose).
	const minLat, maxLat = 45.7, 47.9
	const minLon, maxLon = 5.9, 10.6

	seenIDs := make(map[string]struct{}, len(resp.Results))
	withGeom := 0
	for _, f := range resp.Results {
		require.NotEmpty(t, f.Properties.ObjectID, "every feature should have an objectid")
		_, dup := seenIDs[f.Properties.ObjectID]
		require.False(t, dup, "duplicate objectid %s within one fetch cycle", f.Properties.ObjectID)
		seenIDs[f.Properties.ObjectID] = struct{}{}

		if f.Geometry == nil {
			continue
		}
		withGeom++
		g := f.Geometry.Geometry()
		require.NotNil(t, g, "feature %s has nil orb geometry", f.Properties.ObjectID)
		min := g.Bound().Min
		require.GreaterOrEqual(t, min.Lat(), minLat, "feature %s lat out of range: %v", f.Properties.ObjectID, min)
		require.LessOrEqual(t, min.Lat(), maxLat, "feature %s lat out of range: %v", f.Properties.ObjectID, min)
		require.GreaterOrEqual(t, min.Lon(), minLon, "feature %s lon out of range: %v", f.Properties.ObjectID, min)
		require.LessOrEqual(t, min.Lon(), maxLon, "feature %s lon out of range: %v", f.Properties.ObjectID, min)
	}
	require.NotZero(t, withGeom, "expected at least one feature with geometry")
}
