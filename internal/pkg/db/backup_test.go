package db_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

func TestBackup(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	ctx := logg.WithTestLogger(t.Context(), t)

	d, err := db.Open(ctx, dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })

	backup := db.NewBackup(d, dbPath)
	err = backup.Run(ctx, db.BackupArgs{})
	require.NoError(t, err)

	backupPath := dbPath + ".backup"
	info, err := os.Stat(backupPath)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))

	// Temp file should not exist.
	_, err = os.Stat(backupPath + ".tmp")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestBackup_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	ctx := logg.WithTestLogger(t.Context(), t)

	d, err := db.Open(ctx, dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })

	backup := db.NewBackup(d, dbPath)

	// Run backup twice; second run should overwrite the first.
	err = backup.Run(ctx, db.BackupArgs{})
	require.NoError(t, err)

	err = backup.Run(ctx, db.BackupArgs{})
	require.NoError(t, err)

	backupPath := dbPath + ".backup"
	info, err := os.Stat(backupPath)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
}
