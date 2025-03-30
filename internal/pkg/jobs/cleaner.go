package jobs

import (
	"context"
	"time"

	"github.com/jo-m/goweb/internal/pkg/db"
	"github.com/jo-m/goweb/internal/pkg/logg"
)

// Builtin periodic job to cleanup old jobs from the database.

const jobNameCleaner = "_jobs.cleaner"

type cleanerArgs struct {
	MinAge time.Duration
}

var _ Args = (*cleanerArgs)(nil)

// Kind implements Args.
func (a cleanerArgs) Kind() string { return jobNameCleaner }

type cleaner struct {
	d *db.DB
}

var _ Job[cleanerArgs] = (*cleaner)(nil)

// Run implements Job.
func (c *cleaner) Run(ctx context.Context, args cleanerArgs) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	maxFinishedAt := time.Now().Add(args.MinAge)
	n, err := c.d.QueryRW().CleanupJobs(ctx, maxFinishedAt)
	if n > 0 {
		logg.Debug(ctx, "Cleaned up jobs", "count", n)
	}
	return err
}
