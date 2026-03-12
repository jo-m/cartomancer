//go:build online

package forecast_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/forecast"
)

// TestFetchVariables verifies that the parameter CSV can be fetched and parsed.
func TestFetchVariables(t *testing.T) {
	vars, err := forecast.FetchVariables(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, vars)

	var found bool
	for _, v := range vars {
		if v.Parameter == "T_2M" {
			found = true
			require.Equal(t, "2m Temperature", v.LongName)
			require.Equal(t, "K", v.Unit)
			break
		}
	}
	require.True(t, found, "variable T_2M not found in parameter CSV")
}

// TestDownload verifies that GRIB2 files for a small variable selection can be
// fetched from the newest forecast run and staged in a temporary directory.
func TestDownload(t *testing.T) {
	result, err := forecast.Download(context.Background(), []string{"T_2M", "TOT_PREC"})
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(result.Dir) })

	require.NotEmpty(t, result.Dir)
	require.NotEmpty(t, result.Files)
	require.False(t, result.ReferenceTime.IsZero(), "ReferenceTime must be set")

	for _, f := range result.Files {
		t.Logf("downloaded %s horizon=%s perturbed=%v -> %s",
			f.Variable, f.Horizon, f.Perturbed, f.Path)
		require.NotEmpty(t, f.Variable)
		require.GreaterOrEqual(t, f.Horizon, time.Duration(0))
		require.False(t, f.ValidTime.IsZero())

		info, err := os.Stat(f.Path)
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(0), "file %s should not be empty", f.Path)
	}

	info, err := os.Stat(result.GridConstantsPath)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0), "grid constants file should not be empty")

	info, err = os.Stat(result.VariablesCSVPath)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0), "variables CSV should not be empty")
}
