package trackgroup

import (
	"context"
	"time"

	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

type grouperArgs struct{}

// Kind implements [jobs.Args].
func (a grouperArgs) Kind() string { return "trackgroup.grouper" }

var _ jobs.Args = (*grouperArgs)(nil)

// GrouperArgs returns the args for [Grouper].
//
//revive:disable:unexported-return
func GrouperArgs() grouperArgs {
	return grouperArgs{}
}

// Grouper is a periodic job that groups tracks for every user.
// Use [NewGrouper] to create a new instance.
type Grouper struct {
	d *db.DB
}

// NewGrouper creates a new [Grouper] instance.
func NewGrouper(d *db.DB) *Grouper {
	return &Grouper{d: d}
}

var _ jobs.Job[grouperArgs] = (*Grouper)(nil)

// Run implements [jobs.Job]. It iterates over all users and groups their tracks.
// Individual user failures are logged and skipped so that one broken user does
// not block the rest.
func (g *Grouper) Run(ctx context.Context, _ grouperArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	rows, err := g.d.QueryRO().ListUserUUIDs(ctx)
	if err != nil {
		return err
	}

	for _, userID := range rows {
		if err := GroupUser(ctx, g.d, userID); err != nil {
			logg.Error(ctx, "Failed to group tracks for user, skipping.", "userID", userID, "err", err)
		}
	}

	return nil
}
