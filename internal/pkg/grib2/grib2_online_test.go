//go:build online

package grib2_test

import (
	"bytes"
	"context"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/grib2"
	"jo-m.ch/go/detour/internal/pkg/meteo"
	"jo-m.ch/go/detour/internal/pkg/meteo/vars"
)

// TestOnlineParseU downloads the U wind component at the 0-hour horizon, parses the
// resulting GRIB2 file, verifies that multiple vertical levels are present and
// that the decoded zonal wind values are physically plausible, then renders the
// lowest-level field to a JPEG for visual inspection.
func TestOnlineParseU(t *testing.T) {
	ctx := context.Background()
	result, err := meteo.Download(ctx, []vars.Variable{vars.VarU}, 0, false)
	require.NoError(t, err)
	defer os.RemoveAll(result.Dir)

	require.NotEmpty(t, result.Files, "expected at least one downloaded U file")
	msgs := parseFilePath(t, filepath.Join(result.Dir, result.Files[0].Path))
	require.Greater(t, len(msgs), 1, "multiple vertical levels expected")

	m := msgs[0]
	require.Equal(t, grib2.ParamU, m.Param())
	require.Equal(t, m.ReferenceTime, m.ValidTime) // 0-hour lead time
	require.Len(t, m.Values, 1_147_980)
	for _, v := range m.Values {
		require.False(t, math.IsNaN(float64(v)), "unexpected NaN in U")
		require.Greater(t, v, float32(-100), "U value out of range")
		require.Less(t, v, float32(100), "U value out of range")
	}

	// Parse horizontal grid constants from the download.
	gf, err := os.Open(filepath.Join(result.Dir, result.GridConstantsPath))
	require.NoError(t, err)
	defer gf.Close()
	g, err := grib2.ParseGrid(gf)
	require.NoError(t, err)

	// Render the lowest-level U field to JPEG for visual inspection.
	img := renderMessage(g, m, divergingColor)
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}))
	require.Greater(t, buf.Len(), 0)

	err = os.WriteFile("testdata/TestOnlineParseU.jpeg", buf.Bytes(), 0644)
	require.NoError(t, err)
}

// TestOnlineParseV downloads the V wind component at the 0-hour horizon, parses the
// resulting GRIB2 file, verifies that multiple vertical levels are present and
// that the decoded meridional wind values are physically plausible, then renders the
// lowest-level field to a JPEG for visual inspection.
func TestOnlineParseV(t *testing.T) {
	ctx := context.Background()
	// Forecast runs every 6h and horizon is 33h, so with 8h horizon we always get some files.
	result, err := meteo.Download(ctx, []vars.Variable{vars.VarV}, 0, false)
	require.NoError(t, err)
	defer os.RemoveAll(result.Dir)

	require.NotEmpty(t, result.Files, "expected at least one downloaded V file")
	msgs := parseFilePath(t, filepath.Join(result.Dir, result.Files[0].Path))
	require.Greater(t, len(msgs), 1, "multiple vertical levels expected")

	m := msgs[0]
	require.Equal(t, grib2.ParamV, m.Param())
	require.Equal(t, m.ReferenceTime, m.ValidTime) // 0-hour lead time
	require.Len(t, m.Values, 1_147_980)
	for _, v := range m.Values {
		require.False(t, math.IsNaN(float64(v)), "unexpected NaN in V")
		require.Greater(t, v, float32(-100), "V value out of range")
		require.Less(t, v, float32(100), "V value out of range")
	}

	// Parse horizontal grid constants from the download.
	gf, err := os.Open(filepath.Join(result.Dir, result.GridConstantsPath))
	require.NoError(t, err)
	defer gf.Close()
	g, err := grib2.ParseGrid(gf)
	require.NoError(t, err)

	// Render the lowest-level V field to JPEG for visual inspection.
	img := renderMessage(g, m, divergingColor)
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}))
	require.Greater(t, buf.Len(), 0)

	err = os.WriteFile("testdata/TestOnlineParseV.jpeg", buf.Bytes(), 0644)
	require.NoError(t, err)
}
