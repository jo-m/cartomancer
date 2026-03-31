//go:build online

package meteo_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo/vars"
)

// TestOnlineDownload verifies that GRIB2 files for a small variable selection can be
// fetched from the newest forecast run and staged in a temporary directory.
func TestOnlineDownload(t *testing.T) {
	logger := logg.New(logg.LoggConfig{LogPretty: false, LogLevel: logg.LevelTrace})
	slog.SetDefault(logger)

	result, err := meteo.Download(context.Background(), []vars.Variable{vars.VarT2m, vars.VarTotPr}, 0, false)
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(result.Dir) })

	require.NotEmpty(t, result.Dir)
	require.NotEmpty(t, result.Files)
	require.False(t, result.ReferenceTime.IsZero(), "ReferenceTime must be set")

	for _, f := range result.Files {
		absPath := filepath.Join(result.Dir, f.Path)
		t.Logf("downloaded %s horizon=%s perturbed=%v -> %s",
			f.Meta.Variable, f.Meta.Horizon, f.Meta.Perturbed, absPath)
		require.NotEmpty(t, f.Meta.Variable)
		require.GreaterOrEqual(t, f.Meta.Horizon, time.Duration(0))
		require.False(t, f.Meta.ValidTime.IsZero())

		info, err := os.Stat(absPath)
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(0), "file %s should not be empty", absPath)
	}

	info, err := os.Stat(filepath.Join(result.Dir, result.GridConstantsPath))
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0), "grid constants file should not be empty")
}
