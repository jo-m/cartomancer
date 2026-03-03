// Package track deals with tracks.
package track

import (
	"math"
	"time"
)

type Point struct {
	Time      time.Time
	Lat, Lon  float64
	Elevation float64
}

// MetersTo computes distance in meters to another point.
// TODO: Use `func GreatCircleDistanceM(a, b LatLng) float64`
func (p *Point) MetersTo(other *Point) float64 {
	// https://www.movable-type.co.uk/scripts/latlong.html

	const R = 6371e3              // metres
	phi0 := p.Lat * math.Pi / 180 // φ, λ in radians
	phi1 := other.Lat * math.Pi / 180
	dPhi := (other.Lat - p.Lat) * math.Pi / 180
	dLambda := (other.Lon - p.Lon) * math.Pi / 180

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi0)*math.Cos(phi1)*
			math.Sin(dLambda/2)*math.Sin(dLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// We ignore elevation for distance calculation because it is unreliable
	// and in practice is negligible (e.g. ca. 0.5% at 10% grade).
	return R * c
}

func (p *Point) Sub(other *Point) Point {
	return Point{
		Lat:       p.Lat - other.Lat,
		Lon:       p.Lon - other.Lon,
		Elevation: p.Elevation - other.Elevation,
	}
}

func (p *Point) Add(other *Point) Point {
	return Point{
		Lat:       p.Lat + other.Lat,
		Lon:       p.Lon + other.Lon,
		Elevation: p.Elevation + other.Elevation,
	}
}

func (p *Point) Mul(x float64) Point {
	return Point{
		Lat:       p.Lat * x,
		Lon:       p.Lon * x,
		Elevation: p.Elevation * x,
	}
}

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

type Points []Point
