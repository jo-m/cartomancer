package grib2_test

import (
	"bytes"
	"image/jpeg"
	"math"
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
)

// TestParseHorizontalConstants verifies that ParseGrid extracts lat/lon arrays
// from the horizontal-constants GRIB2 file and that the coordinates are
// plausible for the ICON-CH1 domain.
func TestParseHorizontalConstants(t *testing.T) {
	g := parseGridFile(t)

	require.Len(t, g.Lats, 1_147_980)
	require.Len(t, g.Lons, 1_147_980)

	for i, lat := range g.Lats {
		require.GreaterOrEqual(t, lat, float32(42.0), "lat[%d] out of domain", i)
		require.LessOrEqual(t, lat, float32(51.0), "lat[%d] out of domain", i)
	}
	for i, lon := range g.Lons {
		require.GreaterOrEqual(t, lon, float32(-1.0), "lon[%d] out of domain", i)
		require.LessOrEqual(t, lon, float32(18.0), "lon[%d] out of domain", i)
	}
}

// TestNearestIndex verifies that the spatial index returns the grid point
// closest to well-known Swiss cities.
func TestNearestIndex(t *testing.T) {
	g := parseGridFile(t)

	cases := []struct {
		city    string
		lat     float64
		lon     float64
		maxDist float64 // max acceptable distance in degrees
	}{
		{"Bern", 46.9480, 7.4474, 0.02},
		{"Zurich", 47.3769, 8.5417, 0.02},
		{"Geneva", 46.2044, 6.1432, 0.02},
	}

	for _, tc := range cases {
		t.Run(tc.city, func(t *testing.T) {
			idx := g.NearestIndex(tc.lat, tc.lon)
			require.GreaterOrEqual(t, idx, 0)

			dlat := float64(g.Lats[idx]) - tc.lat
			dlon := float64(g.Lons[idx]) - tc.lon
			dist := math.Sqrt(dlat*dlat + dlon*dlon)
			require.LessOrEqual(t, dist, tc.maxDist,
				"nearest grid point to %s is %.4f° away", tc.city, dist)
		})
	}
}

// TestParseTotPr verifies the TOT_PR file at 0-hour horizon.
// TOT_PR is the total accumulated precipitation (kg/m²) from the reference time.
func TestParseTotPr(t *testing.T) {
	msgs := parseFile(t, "tot_pr_0h.grib2")
	require.Len(t, msgs, 1)

	m := msgs[0]
	// At 0h horizon the valid time equals the reference time.
	require.Equal(t, m.ReferenceTime, m.ValidTime)
	require.Len(t, m.Values, 1_147_980)

	// Accumulated precipitation must be non-negative.
	for _, v := range m.Values {
		require.GreaterOrEqual(t, v, float32(0), "TOT_PR cannot be negative")
	}
}

// TestVisualizeFields renders each test forecast field to a JPEG image and
// verifies the output against a golden snapshot.
func TestVisualizeFields(t *testing.T) {
	g := parseGridFile(t)

	for _, tc := range []struct {
		file string
		cmap colormapFn
	}{
		{"t_2m_0h.grib2", divergingColor},
		{"tot_pr_0h.grib2", sequentialColor},
	} {
		t.Run(tc.file, func(t *testing.T) {
			msgs := parseFile(t, tc.file)
			require.NotEmpty(t, msgs)

			img := renderMessage(g, msgs[0], tc.cmap)
			var buf bytes.Buffer
			err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
			require.NoError(t, err)

			golden.Verify(t, buf.String(), golden.Extension(".jpeg")) // golden.WaitApproval()
		})
	}
}

// TestValueAt checks that ValueAt returns a plausible 2 m temperature at known
// Swiss city locations.  T_2M is stored in Kelvin, so values are expected in
// roughly the 230–320 K range for Switzerland.
func TestValueAt(t *testing.T) {
	g := parseGridFile(t)
	msgs := parseFile(t, "t_2m_0h.grib2")
	require.NotEmpty(t, msgs)
	m := msgs[0]

	cities := []struct{ lat, lon float64 }{
		{46.9480, 7.4474}, // Bern
		{47.3769, 8.5417}, // Zurich
		{46.2044, 6.1432}, // Geneva
	}
	for _, c := range cities {
		v := g.ValueAt(m, c.lat, c.lon)
		require.False(t, math.IsNaN(float64(v)), "ValueAt returned NaN for (%.4f, %.4f)", c.lat, c.lon)
		require.Greater(t, v, float32(200))
		require.Less(t, v, float32(340))
		t.Logf("T_2M at (%.4f, %.4f) = %.2f K", c.lat, c.lon, v)
	}
}
