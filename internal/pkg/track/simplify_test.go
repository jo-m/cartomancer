package track

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimplifyDP(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		require.Nil(t, Points(nil).SimplifyDP(100))
	})

	t.Run("two points", func(t *testing.T) {
		pts := Points{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}}
		got := pts.SimplifyDP(100)
		require.Equal(t, pts, got)
	})

	t.Run("collinear collapses to endpoints", func(t *testing.T) {
		// Three collinear points along the equator.
		pts := Points{
			{Lat: 0, Lon: 0},
			{Lat: 0, Lon: 0.5},
			{Lat: 0, Lon: 1},
		}
		got := pts.SimplifyDP(10)
		require.Len(t, got, 2)
		require.Equal(t, pts[0], got[0])
		require.Equal(t, pts[2], got[1])
	})

	t.Run("zero epsilon keeps all points", func(t *testing.T) {
		pts := Points{
			{Lat: 0, Lon: 0},
			{Lat: 0.001, Lon: 0.5},
			{Lat: 0, Lon: 1},
		}
		got := pts.SimplifyDP(0)
		require.Len(t, got, len(pts))
	})

	t.Run("preserves switchbacks", func(t *testing.T) {
		// Zig-zag pattern: each peak is well above 200m from any baseline.
		// 0.01 deg lat ~ 1.1km, well above the 200m epsilon.
		pts := Points{
			{Lat: 0.00, Lon: 0.00},
			{Lat: 0.01, Lon: 0.01},
			{Lat: 0.00, Lon: 0.02},
			{Lat: 0.01, Lon: 0.03},
			{Lat: 0.00, Lon: 0.04},
		}
		got := pts.SimplifyDP(200)
		require.Equal(t, len(pts), len(got))
	})

	t.Run("first and last always preserved", func(t *testing.T) {
		pts := Points{
			{Lat: 0, Lon: 0},
			{Lat: 0.0001, Lon: 0.5},
			{Lat: 0.0002, Lon: 1.0},
		}
		got := pts.SimplifyDP(1000)
		require.GreaterOrEqual(t, len(got), 2)
		require.Equal(t, pts[0], got[0])
		require.Equal(t, pts[len(pts)-1], got[len(got)-1])
	})

	t.Run("real GPX is reduced", func(t *testing.T) {
		pts := loadGPXPoints(t, "../load/testdata/COURSE_436298480.gpx")
		got := pts.SimplifyDP(PreviewPolylineEpsilon50M)
		require.Less(t, len(got), len(pts))
		require.GreaterOrEqual(t, len(got), 2)
		require.Equal(t, pts[0], got[0])
		require.Equal(t, pts[len(pts)-1], got[len(got)-1])
	})
}

func TestSimplifyForView(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		require.Nil(t, Points(nil).SimplifyForView(5, 25))
	})

	t.Run("two points pass through", func(t *testing.T) {
		pts := Points{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}}
		got := pts.SimplifyForView(5, 25)
		require.Equal(t, pts, got)
	})

	t.Run("dense recording is thinned", func(t *testing.T) {
		// 0.0001 deg ~ 11 m spacing along the equator; 200 such points span ~2.2 km.
		pts := make(Points, 200)
		for i := range pts {
			pts[i] = Point{Lat: 0, Lon: float64(i) * 0.0001}
		}
		got := pts.SimplifyForView(5, 25)
		require.Less(t, len(got), len(pts))
		require.GreaterOrEqual(t, len(got), 2)
		require.Equal(t, pts[0], got[0])
		require.Equal(t, pts[len(pts)-1], got[len(got)-1])
	})

	t.Run("real GPX is reduced", func(t *testing.T) {
		pts := loadGPXPoints(t, "../load/testdata/COURSE_436298480.gpx")
		got := pts.SimplifyForView(PointsViewerEpsilonM, PointsViewerMinDistM)
		require.Less(t, len(got), len(pts))
		require.GreaterOrEqual(t, len(got), 2)
		require.Equal(t, pts[0], got[0])
		require.Equal(t, pts[len(pts)-1], got[len(got)-1])
	})
}

func TestPerpendicularDistanceM(t *testing.T) {
	// Build a 1km east-west segment near the equator.
	a := Point{Lat: 0, Lon: 0}
	b := Point{Lat: 0, Lon: 0.01} // ~1.11km

	t.Run("on the segment", func(t *testing.T) {
		mid := Point{Lat: 0, Lon: 0.005}
		d := perpendicularDistanceM(&mid, &a, &b)
		require.InDelta(t, 0, d, 0.5)
	})

	t.Run("perpendicular offset", func(t *testing.T) {
		// 0.001 deg latitude ~ 111m.
		off := Point{Lat: 0.001, Lon: 0.005}
		d := perpendicularDistanceM(&off, &a, &b)
		require.InDelta(t, 111, d, 1)
	})

	t.Run("zero-length segment uses point distance", func(t *testing.T) {
		p := Point{Lat: 0.001, Lon: 0}
		d := perpendicularDistanceM(&p, &a, &a)
		require.InDelta(t, 111, d, 1)
	})
}
