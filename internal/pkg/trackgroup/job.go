package trackgroup

import (
	"context"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

// GrouperArgs are the arguments for the [Grouper] job.
type GrouperArgs struct {
	UserID string
}

// Kind implements [jobs.Args].
func (a GrouperArgs) Kind() string { return "trackgroup.grouper" }

var _ jobs.Args = (*GrouperArgs)(nil)

// Grouper is a job that groups tracks for a single user.
// Submit it with [jobs.Params.Debounce] enabled so that rapid uploads
// are coalesced into a single grouping run.
// Use [NewGrouper] to create a new instance.
type Grouper struct {
	d *db.DB
}

// NewGrouper creates a new [Grouper] instance.
func NewGrouper(d *db.DB) *Grouper {
	return &Grouper{d: d}
}

var _ jobs.Job[GrouperArgs] = (*Grouper)(nil)

// Run implements [jobs.Job]. It groups tracks for the user specified in args.
func (g *Grouper) Run(ctx context.Context, args GrouperArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	logg.Info(ctx, "track grouping job started", "userID", args.UserID)

	err := GroupUser(ctx, g.d, args.UserID)
	if err != nil {
		return err
	}

	logg.Info(ctx, "track grouping job finished", "userID", args.UserID)
	return nil
}
