// Package track deals with tracks.
package track

import (
	"time"

	"github.com/uber/h3-go/v4"
)

type Point struct {
	Time      time.Time
	Lat, Lon  float64
	Elevation float64
}

// MetersTo computes the great-circle distance in meters to another point.
// Elevation is ignored because it is unreliable and negligible for our purposes (ca. 0.5% at 10% grade).
func (p *Point) MetersTo(other *Point) float64 {
	return h3.GreatCircleDistanceM(
		h3.LatLng{Lat: p.Lat, Lng: p.Lon},
		h3.LatLng{Lat: other.Lat, Lng: other.Lon},
	)
}

// Sub returns p - other (lat, lon, elevation).
func (p *Point) Sub(other *Point) Point {
	return Point{
		Lat:       p.Lat - other.Lat,
		Lon:       p.Lon - other.Lon,
		Elevation: p.Elevation - other.Elevation,
	}
}

// Add returns p + other (lat, lon, elevation).
func (p *Point) Add(other *Point) Point {
	return Point{
		Lat:       p.Lat + other.Lat,
		Lon:       p.Lon + other.Lon,
		Elevation: p.Elevation + other.Elevation,
	}
}

// Mul scales lat, lon, and elevation by x.
func (p *Point) Mul(x float64) Point {
	return Point{
		Lat:       p.Lat * x,
		Lon:       p.Lon * x,
		Elevation: p.Elevation * x,
	}
}

// Interpolate returns the point at fraction x (0-1) between p and other.
func (p *Point) Interpolate(other *Point, x float64) Point {
	if x > 1 {
		panic("x cannot be > 1")
	}
	if x < 0 {
		panic("x cannot be < 0")
	}

	d := other.Sub(p)
	dx := d.Mul(x)
	return p.Add(&dx)
}

// Cell returns the H3 cell containing p at the given resolution.
func (p *Point) Cell(resolution int) h3.Cell {
	latLng := h3.LatLng{
		Lat: p.Lat,
		Lng: p.Lon,
	}
	cell, err := h3.LatLngToCell(latLng, resolution)
	if err != nil {
		panic(err)
	}
	return cell
}

type Points []Point
