//go:build online

package forecast_test

import (
	"context"
	"os"
	"testing"

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

	for key, path := range result.Files {
		t.Logf("downloaded %s -> %s", key, path)
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(0), "file %s should not be empty", path)
	}
}
