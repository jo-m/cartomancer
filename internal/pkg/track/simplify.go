package track

import "math"

// Douglas-Peucker tolerances used when computing preview polylines for
// tracks. The 5 m tolerance retains enough detail for fullscreen track
// rendering; the 50 m tolerance produces tiny outlines suitable for the
// many-tracks map overview where a single track is only a few pixels wide.
const (
	PreviewPolylineEpsilon5M  = 5.0
	PreviewPolylineEpsilon50M = 50.0
)

// Viewer-resolution parameters used by [Points.SimplifyForView] for the
// per-track detail endpoints. The DP epsilon controls how aggressively
// near-collinear points are dropped; the min-distance threshold caps the
// resulting density on dense recordings.
//
// PointsViewerEpsilonM / PointsViewerMinDistM target the track detail page
// (map polyline plus elevation chart). The 25 m floor caps the count for
// 1 Hz FIT files at slow speeds without losing visible detail.
//
// ForecastViewerEpsilonM / ForecastViewerMinDistM target the forecast time
// series. The 500 m spacing roughly matches the ~1.1 km native ICON-CH1-EPS
// grid spacing.
const (
	PointsViewerEpsilonM   = 5.0
	PointsViewerMinDistM   = 25.0
	ForecastViewerEpsilonM = 50.0
	ForecastViewerMinDistM = 500.0
)

// SimplifyForView returns a sparse view of pts suitable for client-side
// rendering: first DP-simplified by epsilonM metres of perpendicular
// distance, then thinned so consecutive points are at least minDistM metres
// apart along the track. The first and last points are always kept.
//
// Both passes are no-ops for non-positive parameters; for an empty or
// single-point input the original slice is returned.
func (pts Points) SimplifyForView(epsilonM, minDistM float64) Points {
	return pts.SimplifyDP(epsilonM).Subsample(minDistM)
}

// SimplifyDP applies the Douglas-Peucker algorithm to pts and returns a
// simplified polyline whose points lie within epsilonM metres of the
// original. The first and last points are always kept.
// epsilonM must be positive; for non-positive values the original slice
// is returned unchanged. Returns nil for nil input.
func (pts Points) SimplifyDP(epsilonM float64) Points {
	n := len(pts)
	if n <= 2 || epsilonM <= 0 {
		if n == 0 {
			return pts
		}
		out := make(Points, n)
		copy(out, pts)
		return out
	}

	// keep[i] reports whether pts[i] is retained.
	keep := make([]bool, n)
	keep[0] = true
	keep[n-1] = true

	// Iterative DP using an explicit stack of (start, end) ranges.
	type span struct{ lo, hi int }
	stack := []span{{0, n - 1}}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if s.hi-s.lo < 2 {
			continue
		}

		idx, dist := farthestPoint(pts, s.lo, s.hi)
		if dist > epsilonM {
			keep[idx] = true
			stack = append(stack, span{s.lo, idx}, span{idx, s.hi})
		}
	}

	out := make(Points, 0, n)
	for i, k := range keep {
		if k {
			out = append(out, pts[i])
		}
	}
	return out
}

// farthestPoint returns the index in (lo, hi) of the point with the greatest
// perpendicular distance, in metres, from the segment (pts[lo], pts[hi]),
// together with that distance.
func farthestPoint(pts Points, lo, hi int) (int, float64) {
	a, b := &pts[lo], &pts[hi]
	bestIdx, bestDist := lo+1, -1.0
	for i := lo + 1; i < hi; i++ {
		d := perpendicularDistanceM(&pts[i], a, b)
		if d > bestDist {
			bestDist = d
			bestIdx = i
		}
	}
	return bestIdx, bestDist
}

// perpendicularDistanceM returns the perpendicular distance in metres from
// p to the great-circle segment between a and b, using a local
// equirectangular projection at a's latitude. This is accurate to well
// under a metre for the segment lengths produced by typical GPS tracks
// and is much cheaper than full geodesic math.
func perpendicularDistanceM(p, a, b *Point) float64 {
	const metersPerDegLat = 111320.0
	cosLat := math.Cos(a.Lat * math.Pi / 180)

	// Convert (lat, lon) deltas to local metres relative to a.
	ax, ay := 0.0, 0.0
	bx := (b.Lon - a.Lon) * metersPerDegLat * cosLat
	by := (b.Lat - a.Lat) * metersPerDegLat
	px := (p.Lon - a.Lon) * metersPerDegLat * cosLat
	py := (p.Lat - a.Lat) * metersPerDegLat

	dx, dy := bx-ax, by-ay
	segLenSq := dx*dx + dy*dy
	if segLenSq == 0 {
		return math.Hypot(px, py)
	}
	// Perpendicular distance from p to the infinite line through a,b.
	num := math.Abs(dy*px - dx*py)
	return num / math.Sqrt(segLenSq)
}
