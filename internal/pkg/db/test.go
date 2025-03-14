package db

import (
	"context"
	"goweb/internal/pkg/logg"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// GetTestDB returns a new temporary test database.
// You must call `Close()` on the returned `DB` when done.
func GetTestDB(t *testing.T) *DB {
	dir := t.TempDir()
	ctx := logg.WithDiscardHandler(context.Background())
	d, err := Open(ctx, filepath.Join(dir, "db"))
	assert.NoError(t, err)
	return d
}
