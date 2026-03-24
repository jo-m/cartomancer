package roadclosures

import (
	"encoding/json"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/require"
)

func mustGeoJSON(t *testing.T, g orb.Geometry) string {
	t.Helper()
	geom := geojson.NewGeometry(g)
	data, err := json.Marshal(geom)
	require.NoError(t, err)
	return string(data)
}

func TestIntersects_Overlapping(t *testing.T) {
	ls := orb.LineString{{8.5, 47.3}, {8.51, 47.31}}
	geomJSON := mustGeoJSON(t, ls)

	// Track points right on the line.
	lats := []float64{47.3, 47.305, 47.31}
	lons := []float64{8.5, 8.505, 8.51}

	require.True(t, Intersects(geomJSON, lats, lons))
}

func TestIntersects_NoOverlap(t *testing.T) {
	ls := orb.LineString{{8.5, 47.3}, {8.51, 47.31}}
	geomJSON := mustGeoJSON(t, ls)

	// Track points far away.
	lats := []float64{46.0, 46.01}
	lons := []float64{7.0, 7.01}

	require.False(t, Intersects(geomJSON, lats, lons))
}

func TestIntersects_InvalidJSON(t *testing.T) {
	require.False(t, Intersects("not json", []float64{47.3}, []float64{8.5}))
}

func TestIntersects_EmptyTrack(t *testing.T) {
	ls := orb.LineString{{8.5, 47.3}, {8.51, 47.31}}
	geomJSON := mustGeoJSON(t, ls)
	require.False(t, Intersects(geomJSON, nil, nil))
}
