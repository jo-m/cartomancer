package session

import (
	"context"
	"time"

	"github.com/jo-m/goweb/internal/pkg/db"
	"github.com/jo-m/goweb/internal/pkg/jobs"
	"github.com/jo-m/goweb/internal/pkg/logg"
)

type cleanerArgs struct {
	IdleTimeout     time.Duration
	AbsoluteTimeout time.Duration
}

// Kind implements [jobs.Args].
func (a cleanerArgs) Kind() string { return "sessions.cleaner" }

var _ jobs.Args = (*cleanerArgs)(nil)

// Cleaner implements a session cleanup job.
// Use [NewCleaner] to create a new instance.
type Cleaner struct {
	d *db.DB
}

// NewCleaner creates a new [Cleaner] instance.
func NewCleaner(d *db.DB) *Cleaner {
	return &Cleaner{d: d}
}

var _ jobs.Job[cleanerArgs] = (*Cleaner)(nil)

// Run implements [jobs.Job].
func (c *Cleaner) Run(ctx context.Context, args cleanerArgs) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	now := time.Now()
	n, err := c.d.QueryRW().CleanupSessions(ctx, db.CleanupSessionsParams{
		CreatedBefore: now.Add(-args.AbsoluteTimeout),
		ActiveBefore:  now.Add(-args.IdleTimeout),
	})
	if n > 0 {
		logg.Info(ctx, "Cleaned up sessions", "count", n)
	}
	return err
}
