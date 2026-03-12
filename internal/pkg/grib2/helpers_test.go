package grib2_test

// Test helpers.

import (
	"image"
	"image/color"
	"math"
	"os"
	"testing"

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

// parseFilePath opens a file from any path and calls grib2.Parse.
func parseFilePath(t *testing.T, path string) []*grib2.Message {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { f.Close() })
	msgs, err := grib2.Parse(f)
	require.NoError(t, err)
	return msgs
}

// parseFile opens a testdata file and calls grib2.Parse.
func parseFile(t *testing.T, name string) []*grib2.Message {
	return parseFilePath(t, testdata+"/"+name)
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
