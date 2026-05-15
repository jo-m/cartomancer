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

// TestIntersects_PointNear verifies that a track point ~17m from a Point
// closure is matched via the k-ring expansion.
// Coordinates match the SG closure at Bollingen / Uznacherstrasse.
func TestIntersects_PointNear(t *testing.T) {
	pt := orb.Point{8.8940519, 47.2196638}
	geomJSON := mustGeoJSON(t, pt)

	lats := []float64{47.21973852750304}
	lons := []float64{8.894247752022238}
	require.True(t, Intersects(geomJSON, lats, lons))
}

// TestIntersects_PointSparseTrack verifies that a coarsely-sampled track is
// matched when it physically crosses a Point closure between two consecutive
// waypoints. Coordinates match the SG closure at Kirchberg / Turpenriet;
// the interpolated path passes 1.9m from the closure marker.
func TestIntersects_PointSparseTrack(t *testing.T) {
	pt := orb.Point{9.0326978, 47.4037267}
	geomJSON := mustGeoJSON(t, pt)

	// Two waypoints bracketing the closure; neither is within 120m of it.
	lats := []float64{47.40349796665312, 47.40393092768113}
	lons := []float64{9.03093921918404, 9.034279442772467}
	require.True(t, Intersects(geomJSON, lats, lons))
}

// TestIntersects_PointFar verifies that a track point far from a Point closure
// is not matched.
func TestIntersects_PointFar(t *testing.T) {
	pt := orb.Point{8.8940519, 47.2196638}
	geomJSON := mustGeoJSON(t, pt)

	lats := []float64{46.0}
	lons := []float64{7.0}
	require.False(t, Intersects(geomJSON, lats, lons))
}
