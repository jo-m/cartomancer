//go:build online

package sz_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures/sz"
)

// TestOnlineFetch hits the live SZ WFS endpoint, verifies that at least one
// feature comes back, and asserts that decoded geometries are well-formed
// WGS84 coordinates inside Switzerland (catches a forgotten axis swap or a
// wrong layer name).
func TestOnlineFetch(t *testing.T) {
	ctx := context.Background()

	features, err := sz.Fetch(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, features, "expected at least one feature")

	// Switzerland's WGS84 bounding box (loose).
	const minLat, maxLat = 45.7, 47.9
	const minLon, maxLon = 5.9, 10.6

	seenIDs := make(map[string]struct{}, len(features))
	withGeom := 0
	for _, f := range features {
		require.NotEmpty(t, f.SourceID, "every feature should have a source id")
		_, dup := seenIDs[f.SourceID]
		require.False(t, dup, "duplicate source id %q within one fetch cycle", f.SourceID)
		seenIDs[f.SourceID] = struct{}{}

		if f.Geometry == nil {
			continue
		}
		withGeom++
		g := f.Geometry.Geometry()
		require.NotNil(t, g, "feature %s has nil orb geometry", f.SourceID)
		min := g.Bound().Min
		require.GreaterOrEqual(t, min.Lat(), minLat, "feature %s lat out of range: %v", f.SourceID, min)
		require.LessOrEqual(t, min.Lat(), maxLat, "feature %s lat out of range: %v", f.SourceID, min)
		require.GreaterOrEqual(t, min.Lon(), minLon, "feature %s lon out of range: %v", f.SourceID, min)
		require.LessOrEqual(t, min.Lon(), maxLon, "feature %s lon out of range: %v", f.SourceID, min)
	}
	require.NotZero(t, withGeom, "expected at least one feature with geometry")
}
