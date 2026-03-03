package db

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

// GetTestDB returns a new temporary test database.
// You must call [Close] on the returned [DB] when done.
func GetTestDB(t *testing.T) *DB {
	t.Helper()

	dir := t.TempDir()
	ctx := logg.WithDiscardHandler(t.Context())
	d, err := Open(ctx, filepath.Join(dir, "db"))
	require.NoError(t, err)
	return d
}
