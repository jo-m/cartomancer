package jobs

import (
	"context"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logg"
)

// Builtin periodic job to cleanup old jobs from the database.

const jobNameCleaner = "_jobs.cleaner"

type cleanerArgs struct{}

var _ Args = (*cleanerArgs)(nil)

// Kind implements Args.
func (a cleanerArgs) Kind() string { return jobNameCleaner }

type cleaner struct {
	d *db.DB
}

var _ Job[cleanerArgs] = (*cleaner)(nil)

// Run implements Job.
func (c *cleaner) Run(ctx context.Context, args cleanerArgs) error {
	n, err := c.d.QueryRW().CleanupJobs(ctx)
	if n > 0 {
		logg.Debug(ctx, "Cleaned up jobs", "count", n)
	}
	return err
}
