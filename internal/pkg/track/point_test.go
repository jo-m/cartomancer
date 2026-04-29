package track

import (
	"fmt"
	"math"
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
)

var (
	berlin = Point{Lat: 52.52, Lon: 13.405, Elevation: 34}
	paris  = Point{Lat: 48.8566, Lon: 2.3522, Elevation: 35}
)

func TestMetersTo(t *testing.T) {
	d := berlin.MetersTo(&paris)
	// Known distance ~878 km
	require.InDelta(t, 878_000, d, 5_000)

	// Same point -> 0
	require.InDelta(t, 0, berlin.MetersTo(&berlin), 0.001)
}

func TestSub(t *testing.T) {
	a := Point{Lat: 10, Lon: 20, Elevation: 100}
	b := Point{Lat: 3, Lon: 5, Elevation: 40}
	got := a.Sub(&b)
	require.InDelta(t, 7, got.Lat, 1e-9)
	require.InDelta(t, 15, got.Lon, 1e-9)
	require.InDelta(t, 60, got.Elevation, 1e-9)
}

func TestAdd(t *testing.T) {
	a := Point{Lat: 1, Lon: 2, Elevation: 10}
	b := Point{Lat: 3, Lon: 4, Elevation: 5}
	got := a.Add(&b)
	require.InDelta(t, 4, got.Lat, 1e-9)
	require.InDelta(t, 6, got.Lon, 1e-9)
	require.InDelta(t, 15, got.Elevation, 1e-9)
}

func TestMul(t *testing.T) {
	p := Point{Lat: 2, Lon: 4, Elevation: 10}
	got := p.Mul(0.5)
	require.InDelta(t, 1, got.Lat, 1e-9)
	require.InDelta(t, 2, got.Lon, 1e-9)
	require.InDelta(t, 5, got.Elevation, 1e-9)
}

func TestInterpolate(t *testing.T) {
	a := Point{Lat: 0, Lon: 0, Elevation: 0}
	b := Point{Lat: 10, Lon: 20, Elevation: 100}

	mid := a.Interpolate(&b, 0.5)
	require.InDelta(t, 5, mid.Lat, 1e-9)
	require.InDelta(t, 10, mid.Lon, 1e-9)
	require.InDelta(t, 50, mid.Elevation, 1e-9)

	require.InDelta(t, 0, a.Interpolate(&b, 0).Lat, 1e-9)
	require.InDelta(t, 10, a.Interpolate(&b, 1).Lat, 1e-9)
}

func TestInterpolatePanics(t *testing.T) {
	a := Point{}
	b := Point{}
	require.Panics(t, func() { a.Interpolate(&b, 1.1) })
	require.Panics(t, func() { a.Interpolate(&b, -0.1) })
}

func TestCell(t *testing.T) {
	cell := berlin.Cell(5)
	require.NotZero(t, cell)

	// Same point -> same cell
	require.Equal(t, berlin.Cell(5), berlin.Cell(5))

	// Coarser resolution -> different (potentially larger) cell
	coarse := berlin.Cell(3)
	fine := berlin.Cell(7)
	require.NotZero(t, coarse)
	require.NotZero(t, fine)

	// H3 resolution range is 0–15; resolution 5 center should be close to berlin
	center, _ := cell.LatLng()
	distDeg := math.Sqrt(math.Pow(center.Lat-berlin.Lat, 2) + math.Pow(center.Lng-berlin.Lon, 2))
	require.Less(t, distDeg, 1.0)
}

func TestSubsample(t *testing.T) {
	// Helper: create a point at a given latitude on the equator.
	pt := func(lat float64) Point {
		return Point{Lat: lat, Lon: 0}
	}

	t.Run("empty", func(t *testing.T) {
		require.Nil(t, Points(nil).Subsample(100))
	})

	t.Run("single point", func(t *testing.T) {
		pts := Points{pt(0)}
		got := pts.Subsample(100)
		require.Equal(t, pts, got)
	})

	t.Run("two points", func(t *testing.T) {
		pts := Points{pt(0), pt(1)}
		got := pts.Subsample(100)
		require.Equal(t, pts, got)
	})

	t.Run("keeps first and last", func(t *testing.T) {
		// Three points very close together; middle one should be dropped.
		pts := Points{pt(0), pt(0.000001), pt(0.000002)}
		got := pts.Subsample(1000)
		require.Len(t, got, 2)
		require.Equal(t, pts[0], got[0])
		require.Equal(t, pts[2], got[1])
	})

	t.Run("keeps distant points", func(t *testing.T) {
		// Points spaced ~111 km apart (1 degree latitude at equator).
		pts := Points{pt(0), pt(1), pt(2), pt(3)}
		got := pts.Subsample(50_000) // 50 km threshold, all are >111 km apart.
		require.Equal(t, pts, got)
	})

	t.Run("filters close points", func(t *testing.T) {
		// 5 points: 0, 0.001, 0.002, 0.003, 1.0 degrees latitude.
		// At equator, 0.001 deg ~ 111 m. With 200 m threshold, only every other close point is kept.
		pts := Points{pt(0), pt(0.001), pt(0.002), pt(0.003), pt(1)}
		got := pts.Subsample(200)
		require.Equal(t, pt(0), got[0])
		require.Equal(t, pt(1), got[len(got)-1])
		// Middle close points should be thinned out.
		require.Less(t, len(got), len(pts))
	})

	t.Run("uses cumulative distance when populated", func(t *testing.T) {
		// Switchback: three retained DP vertices that lie close together by
		// chord (0.001 deg ~ 111 m apart) but are far apart along the path
		// because the original track zig-zagged. The cumulative Distance
		// reflects the path length, not the chord.
		pts := Points{
			{Lat: 0, Lon: 0, Distance: 0},
			{Lat: 0, Lon: 0.001, Distance: 600},  // 600 m along path, ~111 m by chord.
			{Lat: 0, Lon: 0.002, Distance: 1200}, // 1200 m along path, ~222 m by chord.
			{Lat: 0, Lon: 0.003, Distance: 1800},
			{Lat: 0, Lon: 0.004, Distance: 2400},
		}
		// With a 500 m threshold and chord-only logic, the second point would
		// be dropped (chord ~111 m < 500 m). With distance-aware logic, every
		// vertex satisfies the 500 m gap by stored distance and is kept.
		got := pts.Subsample(500)
		require.Equal(t, len(pts), len(got))
	})

	t.Run("uses cumulative distance to thin dense recordings", func(t *testing.T) {
		// Densely sampled points 100 m apart by stored distance, total 1500 m;
		// with a 500 m threshold the 500 m and 1000 m marks are retained.
		pts := make(Points, 16)
		for i := range pts {
			pts[i] = Point{Lat: 0, Lon: float64(i) * 0.0009, Distance: float64(i) * 100}
		}
		got := pts.Subsample(500)
		require.Equal(t, pts[0], got[0])
		require.Equal(t, pts[len(pts)-1], got[len(got)-1])
		require.Len(t, got, 4)
		require.InDelta(t, 500, got[1].Distance, 1e-9)
		require.InDelta(t, 1000, got[2].Distance, 1e-9)
	})

	t.Run("falls back to chord when distance is unpopulated", func(t *testing.T) {
		// All points have zero Distance: behave exactly like the chord path.
		pts := Points{pt(0), pt(0.001), pt(0.002), pt(0.003), pt(1)}
		got := pts.Subsample(200)
		require.Equal(t, pts[0], got[0])
		require.Equal(t, pts[len(pts)-1], got[len(got)-1])
		require.Less(t, len(got), len(pts))
	})
}

func TestPoints(t *testing.T) {
	tr := &Track{pts: []Point{berlin, paris}}
	pts := tr.Points()
	require.Len(t, pts, 2)
	require.Equal(t, berlin.Lat, pts[0].Lat)
	require.Equal(t, paris.Lat, pts[1].Lat)
}

func TestInterpolateByDistance(t *testing.T) {
	t.Run("nil for fewer than two points", func(t *testing.T) {
		require.Nil(t, Points(nil).InterpolateByDistance(100))
		require.Nil(t, Points{berlin}.InterpolateByDistance(100))
	})

	t.Run("nil for zero interval", func(t *testing.T) {
		require.Nil(t, Points{berlin, paris}.InterpolateByDistance(0))
	})

	t.Run("nil when total distance is zero", func(t *testing.T) {
		// Default Distance values (zero) mean total distance is zero.
		require.Nil(t, Points{berlin, paris}.InterpolateByDistance(100))
	})

	t.Run("first and last always included", func(t *testing.T) {
		pts := Points{
			{Lat: berlin.Lat, Lon: berlin.Lon, Distance: 0},
			{Lat: paris.Lat, Lon: paris.Lon, Distance: 878_000},
		}
		result := pts.InterpolateByDistance(1_000_000) // Larger than the total distance.
		require.GreaterOrEqual(t, len(result), 2)
		require.InDelta(t, 0, result[0].DistanceM, 1e-9)
		require.InDelta(t, berlin.Lat, result[0].Lat, 1e-9)
		require.InDelta(t, berlin.Lon, result[0].Lon, 1e-9)
		last := result[len(result)-1]
		require.InDelta(t, 878_000, last.DistanceM, 1e-9)
		require.InDelta(t, paris.Lat, last.Lat, 1e-9)
		require.InDelta(t, paris.Lon, last.Lon, 1e-9)
	})

	t.Run("spacing", func(t *testing.T) {
		// Three points along the equator, each ~111 km apart by stored distance.
		pts := Points{
			{Lat: 0, Lon: 0, Distance: 0},
			{Lat: 0, Lon: 1, Distance: 111_000},
			{Lat: 0, Lon: 2, Distance: 222_000},
		}
		result := pts.InterpolateByDistance(50_000) // 50 km interval.
		require.Greater(t, len(result), 4)

		// All distances should be monotonically increasing.
		for i := 1; i < len(result); i++ {
			require.Greater(t, result[i].DistanceM, result[i-1].DistanceM)
		}

		// Intermediate points should be spaced at exactly 50 km.
		for i := 1; i < len(result)-1; i++ {
			gap := result[i].DistanceM - result[i-1].DistanceM
			require.InDelta(t, 50_000, gap, 1e-9)
		}

		// The last point's distance must equal the input's last cumulative distance.
		require.InDelta(t, 222_000, result[len(result)-1].DistanceM, 1e-9)
	})

	t.Run("trailing partial segment with coincident lat/lon", func(t *testing.T) {
		// After Douglas-Peucker on a switchback, the final input vertex can
		// share lat/lon with the last emitted interpolated point but sit at
		// a slightly larger cumulative distance. The function must still
		// emit a final point at the true end distance, not silently drop it.
		pts := Points{
			{Lat: 0, Lon: 0, Distance: 0},
			{Lat: 0, Lon: 0.001, Distance: 200},
			{Lat: 0, Lon: 0.002, Distance: 400},
			// Same coords as previously emitted (200 m mark) but a larger
			// stored distance: the trailing 50 m partial segment.
			{Lat: 0, Lon: 0.002, Distance: 450},
		}
		result := pts.InterpolateByDistance(200)

		// The final reported distance must equal the input track's last
		// cumulative distance, even though the lat/lon coincides with the
		// previously emitted point.
		require.InDelta(t, 450, result[len(result)-1].DistanceM, 1e-9)
		require.InDelta(t, 0, result[len(result)-1].Lat, 1e-9)
		require.InDelta(t, 0.002, result[len(result)-1].Lon, 1e-9)

		// Distances must remain strictly increasing.
		for i := 1; i < len(result); i++ {
			require.Greater(t, result[i].DistanceM, result[i-1].DistanceM)
		}
	})

	t.Run("distance reflects original cumulative distance after simplification", func(t *testing.T) {
		// Simulate a Douglas-Peucker simplified polyline where the geometry
		// chord is shorter than the original cumulative distance: two vertices
		// at the same coordinates with non-zero stored distance between them.
		pts := Points{
			{Lat: 0, Lon: 0, Distance: 0},
			{Lat: 0, Lon: 1, Distance: 200_000}, // Stored distance > geographic distance.
		}
		result := pts.InterpolateByDistance(50_000)
		require.Greater(t, len(result), 2)

		// Total reported distance must equal the stored cumulative distance,
		// not the chord length recomputed from coordinates.
		require.InDelta(t, 200_000, result[len(result)-1].DistanceM, 1e-9)
		for i := 1; i < len(result)-1; i++ {
			gap := result[i].DistanceM - result[i-1].DistanceM
			require.InDelta(t, 50_000, gap, 1e-9)
		}
	})
}

func TestComputeElevationBounds(t *testing.T) {
	t.Run("empty track", func(t *testing.T) {
		tr := &Track{}
		lo, hi := tr.computeElevationBounds()
		require.Nil(t, lo)
		require.Nil(t, hi)
	})

	t.Run("single point", func(t *testing.T) {
		tr := &Track{pts: []Point{{Elevation: 500}}}
		lo, hi := tr.computeElevationBounds()
		require.NotNil(t, lo)
		require.NotNil(t, hi)
		require.InDelta(t, 500, *lo, 1e-9)
		require.InDelta(t, 500, *hi, 1e-9)
	})

	t.Run("ascending", func(t *testing.T) {
		tr := &Track{pts: []Point{
			{Elevation: 100},
			{Elevation: 300},
			{Elevation: 200},
		}}
		lo, hi := tr.computeElevationBounds()
		require.InDelta(t, 100, *lo, 1e-9)
		require.InDelta(t, 300, *hi, 1e-9)
	})
}

func TestProfileSVGEmpty(t *testing.T) {
	opts := DefaultPreviewOptions()
	opts.Size = 256

	t.Run("nil", func(t *testing.T) {
		svg := Points(nil).ProfileSVG(opts)
		require.Contains(t, svg, `width="256"`)
		require.Contains(t, svg, `height="64"`)
		require.NotContains(t, svg, "polyline")
	})

	t.Run("one point", func(t *testing.T) {
		svg := Points{berlin}.ProfileSVG(opts)
		require.NotContains(t, svg, "polyline")
	})

	t.Run("two identical points", func(t *testing.T) {
		// Zero distance -> empty SVG.
		svg := Points{berlin, berlin}.ProfileSVG(opts)
		require.NotContains(t, svg, "polyline")
	})

	t.Run("dimensions", func(t *testing.T) {
		// Height must be size/3.
		for _, size := range []int{16, 64, 128, 256, 512} {
			o := opts
			o.Size = size
			svg := Points{berlin, paris}.ProfileSVG(o)
			expected := size / 4
			require.Contains(t, svg, fmt.Sprintf(`height="%d"`, expected))
		}
	})
}

func TestProfileSVGSnapshot(t *testing.T) {
	// Uses a GPX with real elevation data (from the load package test fixtures).
	pts := loadGPXPoints(t, "../load/testdata/COURSE_436298480.gpx")
	opts := DefaultPreviewOptions()
	opts.Size = 256
	svg := pts.ProfileSVG(opts)
	golden.Verify(t, svg, golden.Extension(".svg")) // golden.WaitApproval()
}

func TestPreviewSVGPfanni(t *testing.T) {
	pts := loadGPXPoints(t, "testdata/pfanni highlights.gpx")
	opts := DefaultPreviewOptions()
	opts.Size = 256
	svg := pts.PreviewSVG(opts, nil)
	golden.Verify(t, svg, golden.Extension(".svg")) // golden.WaitApproval()
}

func TestPreviewSVGSee(t *testing.T) {
	pts := loadGPXPoints(t, "testdata/See.gpx")
	opts := DefaultPreviewOptions()
	opts.Size = 256
	svg := pts.PreviewSVG(opts, nil)
	golden.Verify(t, svg, golden.Extension(".svg")) // golden.WaitApproval()
}
