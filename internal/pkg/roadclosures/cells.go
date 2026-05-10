package roadclosures

import (
	"github.com/paulmach/orb"
	"github.com/uber/h3-go/v4"
)

// geometryCells computes the set of H3 cells that cover a given orb geometry
// at the specified resolution. Supports Point, MultiPoint, LineString,
// MultiLineString, Polygon, and MultiPolygon.
func geometryCells(g orb.Geometry, resolution int) map[h3.Cell]struct{} {
	cells := make(map[h3.Cell]struct{})
	addPoints(cells, extractPoints(g), resolution)
	return cells
}

// extractPoints recursively collects all coordinate points from a geometry.
func extractPoints(g orb.Geometry) []orb.Point {
	switch v := g.(type) {
	case orb.Point:
		return []orb.Point{v}
	case orb.MultiPoint:
		return []orb.Point(v)
	case orb.LineString:
		return []orb.Point(v)
	case orb.MultiLineString:
		var pts []orb.Point
		for _, ls := range v {
			pts = append(pts, []orb.Point(ls)...)
		}
		return pts
	case orb.Polygon:
		var pts []orb.Point
		for _, ring := range v {
			pts = append(pts, []orb.Point(ring)...)
		}
		return pts
	case orb.MultiPolygon:
		var pts []orb.Point
		for _, poly := range v {
			for _, ring := range poly {
				pts = append(pts, []orb.Point(ring)...)
			}
		}
		return pts
	default:
		return nil
	}
}

// addPoints converts orb.Points to H3 cells and adds them to the set.
// Adjacent points are interpolated at half the hexagon edge length to avoid
// skipping cells on straight segments.
func addPoints(cells map[h3.Cell]struct{}, pts []orb.Point, resolution int) {
	if len(pts) == 0 {
		return
	}

	edgeLenM, err := h3.HexagonEdgeLengthAvgM(resolution)
	if err != nil {
		panic(err)
	}
	stepM := edgeLenM / 2

	for i, pt := range pts {
		cell, err := h3.LatLngToCell(h3.LatLng{Lat: pt.Lat(), Lng: pt.Lon()}, resolution)
		if err != nil {
			continue
		}
		cells[cell] = struct{}{}

		if i == 0 {
			continue
		}

		prev := pts[i-1]
		distM := h3.GreatCircleDistanceM(
			h3.LatLng{Lat: prev.Lat(), Lng: prev.Lon()},
			h3.LatLng{Lat: pt.Lat(), Lng: pt.Lon()},
		)
		steps := int(distM/stepM + 1)
		for j := 1; j < steps; j++ {
			frac := float64(j) / float64(steps)
			lat := prev.Lat() + frac*(pt.Lat()-prev.Lat())
			lon := prev.Lon() + frac*(pt.Lon()-prev.Lon())
			c, err := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lon}, resolution)
			if err != nil {
				continue
			}
			cells[c] = struct{}{}
		}
	}
}
