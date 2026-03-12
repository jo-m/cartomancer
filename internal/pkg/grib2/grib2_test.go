package grib2_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"testing"
	"time"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/grib2"
)

const (
	snapWidth  = 950
	snapHeight = 450

	// Bounding box of the ICON-CH1 domain in degrees.
	snapMinLat = float32(42.0)
	snapMaxLat = float32(51.0)
	snapMinLon = float32(-1.0)
	snapMaxLon = float32(18.0)
)

const testdata = "testdata"

// parseFile opens a testdata file and calls grib2.Parse.
func parseFile(t *testing.T, name string) []*grib2.Message {
	t.Helper()
	f, err := os.Open(testdata + "/" + name)
	require.NoError(t, err)
	t.Cleanup(func() { f.Close() })
	msgs, err := grib2.Parse(f)
	require.NoError(t, err)
	return msgs
}

// parseGridFile opens the horizontal constants testdata file and returns a Grid.
func parseGridFile(t *testing.T) *grib2.Grid {
	t.Helper()
	f, err := os.Open(testdata + "/horiz_const.grib2")
	require.NoError(t, err)
	t.Cleanup(func() { f.Close() })
	g, err := grib2.ParseGrid(f)
	require.NoError(t, err)
	return g
}

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

// TestParseU parses the U multi-level control forecast file, checks that
// multiple vertical levels are present, and verifies the lowest level's
// decoded zonal wind values are physically plausible.
func TestParseU(t *testing.T) {
	msgs := parseFile(t, "u_0h.grib2")
	require.Greater(t, len(msgs), 1) // multiple vertical levels expected

	m := msgs[0]
	require.Equal(t, grib2.ParamUWind10m, m.Param())
	require.Equal(t, m.ReferenceTime, m.ValidTime) // 0-hour lead time
	require.Len(t, m.Values, 1_147_980)
	for _, v := range m.Values {
		require.False(t, math.IsNaN(float64(v)), "unexpected NaN in U")
		require.Greater(t, v, float32(-100), "U value out of range")
		require.Less(t, v, float32(100), "U value out of range")
	}
}

// TestParseV parses the V multi-level control forecast file, checks that
// multiple vertical levels are present, and verifies the lowest level's
// decoded meridional wind values are physically plausible.
func TestParseV(t *testing.T) {
	msgs := parseFile(t, "v_0h.grib2")
	require.Greater(t, len(msgs), 1) // multiple vertical levels expected

	m := msgs[0]
	require.Equal(t, grib2.ParamVWind10m, m.Param())
	require.Equal(t, m.ReferenceTime, m.ValidTime) // 0-hour lead time
	require.Len(t, m.Values, 1_147_980)
	for _, v := range m.Values {
		require.False(t, math.IsNaN(float64(v)), "unexpected NaN in V")
		require.Greater(t, v, float32(-100), "V value out of range")
		require.Less(t, v, float32(100), "V value out of range")
	}
}

// TestParseTotPR verifies the TOT_PR file at 10-hour lead time.
// TOT_PR is the instantaneous total precipitation rate (kg/m²/s) at t=10h.
func TestParseTotPR(t *testing.T) {
	msgs := parseFile(t, "tot_pr_10h.grib2")
	require.Len(t, msgs, 1)

	m := msgs[0]
	// Valid time for a PDT 1 (instantaneous) field is ref + 10h.
	expected := m.ReferenceTime.Add(10 * time.Hour)
	require.Equal(t, expected, m.ValidTime)
	require.Len(t, m.Values, 1_147_980)

	// Precipitation rate must be non-negative.
	for _, v := range m.Values {
		require.GreaterOrEqual(t, v, float32(0), "TOT_PR cannot be negative")
	}
}

// colormapFn maps a scalar value within [vmin, vmax] to an RGBA colour.
type colormapFn func(v, vmin, vmax float32) color.RGBA

// lerp8 linearly interpolates between two uint8 channel values at position t ∈ [0,1].
func lerp8(a, b uint8, t float32) uint8 {
	return uint8(float32(a)*(1-t) + float32(b)*t)
}

// divergingColor maps values to a blue–white–red palette centred on the
// mid-point of [vmin, vmax].  Useful for signed quantities such as wind speed.
func divergingColor(v, vmin, vmax float32) color.RGBA {
	half := (vmax - vmin) / 2
	if half <= 0 {
		return color.RGBA{128, 128, 128, 255}
	}
	center := vmin + half
	t := (v - center) / half // [-1, 1]
	if t < 0 {
		// blue → white as t goes from -1 → 0
		s := t + 1
		return color.RGBA{lerp8(0, 255, s), lerp8(0, 255, s), 255, 255}
	}
	// white → red as t goes from 0 → 1
	return color.RGBA{255, lerp8(255, 0, t), lerp8(255, 0, t), 255}
}

// sequentialColor maps values to a white–blue palette.  Useful for
// non-negative quantities such as accumulated precipitation.
func sequentialColor(v, vmin, vmax float32) color.RGBA {
	if vmax <= vmin {
		return color.RGBA{255, 255, 255, 255}
	}
	t := (v - vmin) / (vmax - vmin)
	return color.RGBA{lerp8(255, 0, t), lerp8(255, 0, t), 255, 255}
}

// renderMessage scatter-plots all grid points of msg onto a snapWidth×snapHeight
// RGBA image using the supplied colour map.  Points outside the domain bounding
// box are silently skipped.
func renderMessage(g *grib2.Grid, msg *grib2.Message, cmap colormapFn) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, snapWidth, snapHeight))
	bg := color.RGBA{210, 210, 210, 255}
	for y := range snapHeight {
		for x := range snapWidth {
			img.SetRGBA(x, y, bg)
		}
	}

	// Determine value range, ignoring NaN.
	vmin, vmax := float32(math.MaxFloat32), float32(-math.MaxFloat32)
	for _, v := range msg.Values {
		if math.IsNaN(float64(v)) {
			continue
		}
		if v < vmin {
			vmin = v
		}
		if v > vmax {
			vmax = v
		}
	}

	for i, v := range msg.Values {
		if math.IsNaN(float64(v)) {
			continue
		}
		lat := g.Lats[i]
		lon := g.Lons[i]
		x := int((lon - snapMinLon) / (snapMaxLon - snapMinLon) * float32(snapWidth))
		y := int((snapMaxLat - lat) / (snapMaxLat - snapMinLat) * float32(snapHeight))
		if x < 0 || x >= snapWidth || y < 0 || y >= snapHeight {
			continue
		}
		img.SetRGBA(x, y, cmap(v, vmin, vmax))
	}
	return img
}

// TestVisualizeFields renders each test forecast field to a JPEG image and
// verifies the output against a golden snapshot.
func TestVisualizeFields(t *testing.T) {
	g := parseGridFile(t)

	for _, tc := range []struct {
		file string
		cmap colormapFn
	}{
		{"u_0h.grib2", divergingColor},
		{"tot_pr_10h.grib2", sequentialColor},
	} {
		t.Run(tc.file, func(t *testing.T) {
			msgs := parseFile(t, tc.file)
			require.NotEmpty(t, msgs)

			img := renderMessage(g, msgs[0], tc.cmap)
			var buf bytes.Buffer
			err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
			require.NoError(t, err)

			golden.Verify(t, buf.String()) // golden.WaitApproval()
		})
	}
}

// TestValueAt checks that ValueAt returns a plausible wind speed at known
// Swiss city locations using the first level of the U multi-level field.
func TestValueAt(t *testing.T) {
	g := parseGridFile(t)
	msgs := parseFile(t, "u_0h.grib2")
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
		require.Greater(t, v, float32(-100))
		require.Less(t, v, float32(100))
		t.Logf("U at (%.4f, %.4f) = %.3f m/s", c.lat, c.lon, v)
	}
}
