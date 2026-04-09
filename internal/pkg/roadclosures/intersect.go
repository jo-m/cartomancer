package roadclosures

import (
	"encoding/json"
	"log/slog"

	"github.com/paulmach/orb/geojson"
	"github.com/uber/h3-go/v4"
)

// Intersects reports whether a closure geometry intersects any of the given
// track points. Both the closure and the track points are converted to H3 cells
// at [FineResolution]. Track points are not interpolated (sparse), so this is a
// fast approximate check.
func Intersects(closureGeometryJSON string, trackLats, trackLons []float64) bool {
	var geom geojson.Geometry
	if err := json.Unmarshal([]byte(closureGeometryJSON), &geom); err != nil {
		slog.Warn("failed to parse closure geometry JSON", "err", err)
		return false
	}

	closureCells := geometryCells(geom.Geometry(), FineResolution)
	if len(closureCells) == 0 {
		return false
	}

	for i := range trackLats {
		cell, err := h3.LatLngToCell(h3.LatLng{Lat: trackLats[i], Lng: trackLons[i]}, FineResolution)
		if err != nil {
			continue
		}
		if _, ok := closureCells[cell]; ok {
			return true
		}
	}
	return false
}
