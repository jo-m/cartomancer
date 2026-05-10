//go:build online

package zh_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures/zh"
)

func TestOnlineFetch(t *testing.T) {
	ctx := context.Background()

	features, err := zh.Fetch(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, features, "expected at least one feature")

	// Switzerland's WGS84 bounding box (loose).
	const minLat, maxLat = 45.7, 47.9
	const minLon, maxLon = 5.9, 10.6

	withGeom := 0
	for _, f := range features {
		require.NotEmpty(t, f.GMLID)
		require.NotEmpty(t, f.StatusBaustelle)
		if f.Geometry == nil {
			continue
		}
		withGeom++
		// Sample one coordinate of the decoded geometry and make sure it
		// looks like WGS84 inside Switzerland (catches a forgotten axis swap).
		g := f.Geometry.Geometry()
		require.NotNil(t, g, "feature %s has nil orb geometry", f.GMLID)
		require.NotEmpty(t, g.Bound().Min)
		min := g.Bound().Min
		require.GreaterOrEqual(t, min.Lat(), minLat, "feature %s lat out of range: %v", f.GMLID, min)
		require.LessOrEqual(t, min.Lat(), maxLat, "feature %s lat out of range: %v", f.GMLID, min)
		require.GreaterOrEqual(t, min.Lon(), minLon, "feature %s lon out of range: %v", f.GMLID, min)
		require.LessOrEqual(t, min.Lon(), maxLon, "feature %s lon out of range: %v", f.GMLID, min)
	}
	require.NotZero(t, withGeom, "expected at least one feature with geometry")
}
