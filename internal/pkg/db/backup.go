package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"jo-m.ch/go/detour/internal/pkg/logg"
)

const jobNameBackup = "db.backup"

// BackupArgs are the arguments for the periodic database backup job.
type BackupArgs struct{}

// Kind implements [jobs.Args].
func (a BackupArgs) Kind() string { return jobNameBackup }

// Backup is a job handler that vacuums the database and writes a backup file.
// The backup is written to a temporary file first and then atomically renamed
// to ensure a valid file is always present on disk.
type Backup struct {
	d    *DB
	path string
}

// NewBackup creates a new Backup job handler.
//
// Parameters:
//   - d: the database connection used for VACUUM operations.
//   - path: the path of the SQLite database file; the backup will be written to path+".backup".
func NewBackup(d *DB, path string) *Backup {
	return &Backup{d: d, path: path}
}

// Run implements [jobs.Job].
// It first runs VACUUM on the main database to reclaim space, then creates
// a vacuumed backup copy via VACUUM INTO a temporary file, and finally
// renames the temporary file to the final backup path.
func (b *Backup) Run(ctx context.Context, _ BackupArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	logg.Info(ctx, "database backup started")

	// Write a vacuumed copy to a temp file.
	backupPath := b.path + ".backup"
	tmpPath := backupPath + ".tmp"

	_, err := b.d.rw.ExecContext(ctx, "VACUUM INTO ?", tmpPath)
	if err != nil {
		// Clean up partial temp file on failure.
		_ = os.Remove(tmpPath)
		return fmt.Errorf("vacuum into: %w", err)
	}

	// Atomically replace the backup file.
	err = os.Rename(tmpPath, backupPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename backup: %w", err)
	}

	logg.Info(ctx, "database backup completed", "path", backupPath)
	return nil
}
