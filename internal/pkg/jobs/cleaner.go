package jobs

import (
	"context"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logg"
)

// Builtin periodic job to cleanup old jobs from the database.

const jobNameCleanup = "_jobs_cleanup"

type cleanerArgs struct{}

func (a cleanerArgs) Kind() string { return jobNameCleanup }

var _ Args = (*cleanerArgs)(nil)

type cleaner struct{}

var _ Worker[cleanerArgs] = (*cleaner)(nil)

func (c *cleaner) Work(ctx context.Context, d *db.DB, args cleanerArgs) error {
	n, err := d.QueryRW().CleanupJobs(ctx)
	if n > 0 {
		logg.Debug(ctx, "Cleaned up jobs", "count", n)
	}
	return err
}
