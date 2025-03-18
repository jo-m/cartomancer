package session

import (
	"context"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/jobs"
	"goweb/internal/pkg/logg"
	"time"
)

type cleanerArgs struct {
	MaxIdleTimeout     time.Duration
	MaxAbsoluteTimeout time.Duration
}

// Kind implements jobs.Args.
func (a cleanerArgs) Kind() string { return "sessions.cleaner" }

var _ jobs.Args = (*cleanerArgs)(nil)

type Cleaner struct{}

var _ jobs.Job[cleanerArgs] = (*Cleaner)(nil)

// Run implements jobs.Job.
func (c *Cleaner) Run(ctx context.Context, d *db.DB, args cleanerArgs) error {
	now := time.Now()
	n, err := d.QueryRW().CleanupSessions(ctx, db.CleanupSessionsParams{
		CreatedBefore: now.Add(-args.MaxAbsoluteTimeout),
		ActiveBefore:  now.Add(-args.MaxIdleTimeout),
	})
	if n > 0 {
		logg.Info(ctx, "Cleaned up sessions", "count", n)
	}
	return err
}
