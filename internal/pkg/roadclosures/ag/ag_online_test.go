//go:build online

package ag_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures/ag"
)

// TestOnlineFetch hits the live AG ArcGIS MapServer endpoint, verifies that
// at least one feature comes back, and asserts that decoded geometries are
// well-formed WGS84 coordinates inside Switzerland (catches a forgotten
// outSR override or a wrong layer index).
func TestOnlineFetch(t *testing.T) {
	ctx := context.Background()

	fc, err := ag.Fetch(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, fc.Features, "expected at least one feature")

	// Switzerland's WGS84 bounding box (loose).
	const minLat, maxLat = 45.7, 47.9
	const minLon, maxLon = 5.9, 10.6

	seenIDs := make(map[int64]struct{}, len(fc.Features))
	withGeom := 0
	for _, f := range fc.Features {
		require.NotZero(t, f.Properties.ObjectID, "every feature should have an OBJECTID")
		_, dup := seenIDs[f.Properties.ObjectID]
		require.False(t, dup, "duplicate OBJECTID %d within one fetch cycle", f.Properties.ObjectID)
		seenIDs[f.Properties.ObjectID] = struct{}{}

		if f.Geometry == nil {
			continue
		}
		withGeom++
		g := f.Geometry.Geometry()
		require.NotNil(t, g, "feature %d has nil orb geometry", f.Properties.ObjectID)
		min := g.Bound().Min
		require.GreaterOrEqual(t, min.Lat(), minLat, "feature %d lat out of range: %v", f.Properties.ObjectID, min)
		require.LessOrEqual(t, min.Lat(), maxLat, "feature %d lat out of range: %v", f.Properties.ObjectID, min)
		require.GreaterOrEqual(t, min.Lon(), minLon, "feature %d lon out of range: %v", f.Properties.ObjectID, min)
		require.LessOrEqual(t, min.Lon(), maxLon, "feature %d lon out of range: %v", f.Properties.ObjectID, min)
	}
	require.NotZero(t, withGeom, "expected at least one feature with geometry")
}
