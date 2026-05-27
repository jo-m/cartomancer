package forecast

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo/vars"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// constHandle builds a Handle that returns constant direct/diffuse irradiance
// values at all sample times and location indices, for testing the integrator
// without a full forecast database.
func constHandle(direct, diffuse float32, start time.Time, dur time.Duration, nLocs int) *Handle {
	makeEntries := func(v float32) []timedValues {
		vals := make([]float32, nLocs)
		for i := range vals {
			vals[i] = v
		}
		return []timedValues{{
			validTime:      start.Add(-time.Hour),
			validUntilTime: start.Add(dur).Add(time.Hour),
			vals:           vals,
		}}
	}
	return &Handle{
		values: map[string][]timedValues{
			vars.VarAswdirS.Name:  makeEntries(direct),
			vars.VarAswdifdS.Name: makeEntries(diffuse),
		},
	}
}

// linePoints returns a straight line of n points spaced stepM metres apart,
// with cumulative distance populated.
func linePoints(n int, stepM float64) track.Points {
	pts := make(track.Points, n)
	for i := range pts {
		pts[i] = track.Point{
			Lat:      46.0,
			Lon:      8.0 + float64(i)*0.001,
			Distance: float64(i) * stepM,
		}
	}
	return pts
}

func TestComputeSunIntensity_ClearSky(t *testing.T) {
	start := fixedTime()
	pts := linePoints(20, 200) // 20 points, 200 m apart -> ~3.8 km total
	speedMs := 7.78            // ~28 km/h, full ride ~488 s

	h := constHandle(800, 200, start, time.Hour, len(pts))
	got := ComputeSunIntensity(h, pts, start, speedMs)
	require.False(t, math.IsNaN(got.Index))
	require.False(t, math.IsNaN(got.DoseJm2))
	// 1000 W/m^2 * ~488 s = ~4.88e5 J/m^2; scale 1.2e-6 -> ~0.59. A short ride
	// under clear sky should yield a small but nonzero index.
	require.Greater(t, got.Index, sunIntensityMin)
	require.Less(t, got.Index, 1.0)
	require.Greater(t, got.DoseJm2, 0.0)
	require.InEpsilon(t, got.DoseJm2*sunIntensityScale, got.Index, 1e-9)
}

func TestComputeSunIntensity_LongClearSkyRide(t *testing.T) {
	start := fixedTime()
	// 200 points 1 km apart -> 199 km, at 28 km/h -> ~7.1 h.
	pts := linePoints(200, 1000)
	speedMs := 7.78

	h := constHandle(800, 200, start, 12*time.Hour, len(pts))
	got := ComputeSunIntensity(h, pts, start, speedMs)
	require.False(t, math.IsNaN(got.Index))
	require.False(t, math.IsNaN(got.DoseJm2))
	// 1000 W/m^2 * (199000/7.78) s = ~2.56e7 J/m^2, scale 1.2e-6 -> ~30.7 -> clamped to max.
	require.Equal(t, sunIntensityMax, got.Index)
	// Dose is unclamped, so it should exceed the dose corresponding to the max
	// index ([sunIntensityMaxDoseSED] SED of broadband-equivalent).
	require.Greater(t, got.DoseJm2, sunIntensityMax/sunIntensityScale)
}

func TestComputeSunIntensity_BelowThreshold(t *testing.T) {
	start := fixedTime()
	pts := linePoints(20, 1000)
	speedMs := 7.78

	// Total SW = 20 + 20 = 40 W/m^2 < threshold (50). Contributes 0 to dose.
	h := constHandle(20, 20, start, time.Hour, len(pts))
	got := ComputeSunIntensity(h, pts, start, speedMs)
	require.False(t, math.IsNaN(got.Index))
	require.Equal(t, sunIntensityMin, got.Index, "below-threshold samples should yield the floor")
	require.Equal(t, 0.0, got.DoseJm2, "below-threshold samples should yield zero dose")
}

func TestComputeSunIntensity_NoData(t *testing.T) {
	start := fixedTime()
	pts := linePoints(10, 500)
	speedMs := 7.78

	h := &Handle{values: map[string][]timedValues{}}
	got := ComputeSunIntensity(h, pts, start, speedMs)
	require.True(t, math.IsNaN(got.Index), "expected NaN index when no irradiance data is available")
	require.True(t, math.IsNaN(got.DoseJm2), "expected NaN dose when no irradiance data is available")
}

func TestComputeSunIntensity_GuardsInvalidInputs(t *testing.T) {
	start := fixedTime()
	pts := linePoints(10, 500)
	h := constHandle(800, 200, start, time.Hour, len(pts))

	for _, got := range []SunIntensity{
		ComputeSunIntensity(nil, pts, start, 7.78),
		ComputeSunIntensity(h, pts[:1], start, 7.78),
		ComputeSunIntensity(h, pts, start, 0),
		ComputeSunIntensity(h, pts, start, -1),
	} {
		require.True(t, math.IsNaN(got.Index))
		require.True(t, math.IsNaN(got.DoseJm2))
	}
}

func TestComputeSunIntensity_AlwaysWithinBounds(t *testing.T) {
	start := fixedTime()
	speedMs := 7.78

	for _, sw := range []float32{0, 100, 200, 500, 1000, 1500} {
		pts := linePoints(50, 1000)
		h := constHandle(sw*0.8, sw*0.2, start, 12*time.Hour, len(pts))
		got := ComputeSunIntensity(h, pts, start, speedMs)
		require.False(t, math.IsNaN(got.Index))
		require.False(t, math.IsNaN(got.DoseJm2))
		require.GreaterOrEqual(t, got.Index, sunIntensityMin)
		require.LessOrEqual(t, got.Index, sunIntensityMax)
		require.GreaterOrEqual(t, got.DoseJm2, 0.0)
	}
}
