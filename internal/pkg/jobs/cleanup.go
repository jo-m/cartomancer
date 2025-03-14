package jobs

import (
	"context"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logg"
)

// Builtin periodic job to cleanup old jobs from the database.

const jobNameCleanup = "_jobs_cleanup"

type cleanupArgs struct{}

func (a cleanupArgs) Kind() string { return jobNameCleanup }

var _ Args = (*cleanupArgs)(nil)

type cleaner struct{}

var _ Worker[cleanupArgs] = (*cleaner)(nil)

func (c *cleaner) Work(ctx context.Context, d *db.DB, args cleanupArgs) error {
	n, err := d.QueryRW().CleanupJobs(ctx)
	if n > 0 {
		logg.Debug(ctx, "Cleaned up jobs", "count", n)
	}
	return err
}
