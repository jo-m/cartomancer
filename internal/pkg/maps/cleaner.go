package maps

import (
	"context"
	"os"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

const cleanerTimeout = 5 * time.Minute

// CleanerArgs are the arguments for the map cleaner job.
type CleanerArgs struct{}

// Kind implements [jobs.Args].
func (CleanerArgs) Kind() string { return "maps.cleaner" }

var _ jobs.Args = (*CleanerArgs)(nil)

// Cleaner removes PMTiles files and database records for map builds that have been
// marked for deletion by the downloader.
// Use [NewCleaner] to create an instance.
type Cleaner struct {
	d       *db.DB
	mapsDir string
}

// NewCleaner creates a new [Cleaner] instance.
// The mapsDir parameter is the directory where extracted PMTiles files are stored.
func NewCleaner(d *db.DB, mapsDir string) *Cleaner {
	return &Cleaner{d: d, mapsDir: mapsDir}
}

var _ jobs.Job[CleanerArgs] = (*Cleaner)(nil)

// Run implements [jobs.Job].
// It deletes the PMTiles file for each build marked for deletion, then removes the database record.
func (c *Cleaner) Run(ctx context.Context, _ CleanerArgs) error {
	ctx, cancel := context.WithTimeout(ctx, cleanerTimeout)
	defer cancel()

	builds, err := c.d.QueryRO().ListBuildsMarkedForDeletion(ctx)
	if err != nil {
		return err
	}

	for _, b := range builds {
		c.deleteOne(ctx, b)
	}

	return nil
}

// deleteOne removes the PMTiles file for a single build and then its database record.
// Errors are logged but not propagated, so a single failure does not block the rest.
func (c *Cleaner) deleteOne(ctx context.Context, b db.MapBuild) {
	path := OutputPath(c.mapsDir, b.Uuid)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		logg.Error(ctx, "failed to remove map file marked for deletion", "uuid", b.Uuid, "path", path, "err", err)
		return
	}

	if _, err := c.d.QueryRW().DeleteMapBuild(ctx, b.Uuid); err != nil {
		logg.Error(ctx, "failed to delete map build record after file removal", "uuid", b.Uuid, "err", err)
	}

	logg.Info(ctx, "deleted map build marked for deletion", "uuid", b.Uuid, "key", b.Key)
}
