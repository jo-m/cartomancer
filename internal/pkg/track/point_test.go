package track

import (
	"math"
	"testing"

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
