package roadclosures

import (
	"encoding/json"
	"log/slog"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/uber/h3-go/v4"
)

// pointExpandRadius is the H3 GridDisk radius applied to Point closure cells
// during the fine intersection check. At FineResolution (res 12, ~10.8m edge
// length), radius 1 covers roughly 19m from the closure marker, catching tracks
// that pass near (but not exactly through) a point closure marker.
const pointExpandRadius = 1

// BuildTrackCells converts a track's lat/lon slices to an interpolated H3 cell
// set at [FineResolution]. Consecutive points are interpolated at sub-cell
// granularity so that closures falling between sparse GPS fixes are not missed.
// Pre-compute this once and pass it to [IntersectsCells] for each closure.
func BuildTrackCells(lats, lons []float64) map[h3.Cell]struct{} {
	pts := make([]orb.Point, len(lats))
	for i := range lats {
		pts[i] = orb.Point{lons[i], lats[i]}
	}
	cells := make(map[h3.Cell]struct{})
	addPoints(cells, pts, FineResolution)
	return cells
}

// IntersectsCells reports whether a closure geometry intersects the pre-computed
// track cell set produced by [BuildTrackCells]. For Point closures the closure
// cell set is expanded by a k-ring of [pointExpandRadius] to catch tracks that
// pass near (but not exactly through) the closure marker.
func IntersectsCells(closureGeometryJSON string, trackCells map[h3.Cell]struct{}) bool {
	var geom geojson.Geometry
	if err := json.Unmarshal([]byte(closureGeometryJSON), &geom); err != nil {
		slog.Warn("failed to parse closure geometry JSON", "err", err)
		return false
	}

	g := geom.Geometry()
	closureCells := geometryCells(g, FineResolution)
	if len(closureCells) == 0 {
		return false
	}

	// Point closures are markers, not exact paths; expand to a small buffer zone.
	switch g.(type) {
	case orb.Point:
		expanded := make(map[h3.Cell]struct{}, len(closureCells)*(1+3*pointExpandRadius*(pointExpandRadius+1)))
		for cell := range closureCells {
			disk, err := h3.GridDisk(cell, pointExpandRadius)
			if err != nil {
				expanded[cell] = struct{}{}
				continue
			}
			for _, neighbor := range disk {
				expanded[neighbor] = struct{}{}
			}
		}
		closureCells = expanded
	}

	for cell := range trackCells {
		if _, ok := closureCells[cell]; ok {
			return true
		}
	}
	return false
}

// Intersects reports whether a closure geometry intersects any of the given
// track points. It is a convenience wrapper around [BuildTrackCells] and
// [IntersectsCells]; when checking multiple closures against the same track,
// call those two functions directly to avoid recomputing the track cell set.
func Intersects(closureGeometryJSON string, trackLats, trackLons []float64) bool {
	return IntersectsCells(closureGeometryJSON, BuildTrackCells(trackLats, trackLons))
}
