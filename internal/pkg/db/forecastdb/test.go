package forecastdb

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

// GetTestDB returns a new temporary forecast test database with all migrations applied.
// You must call [Close] on the returned [DB] when done.
func GetTestDB(t *testing.T) *DB {
	t.Helper()

	dir := t.TempDir()
	ctx := logg.WithTestLogger(t.Context(), t)
	d, err := Open(ctx, filepath.Join(dir, "forecast.db"))
	require.NoError(t, err)
	return d
}
