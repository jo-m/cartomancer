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

const userBatchSize = 100

// Run implements [jobs.Job]. It iterates over all users using cursor-based
// pagination and groups their tracks. Individual user failures are logged and
// skipped so that one broken user does not block the rest.
func (g *Grouper) Run(ctx context.Context, _ grouperArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	logg.Info(ctx, "Track grouping job started.")

	cursor := ""
	for {
		batch, err := g.d.QueryRO().ListUserUUIDsAfter(ctx, db.ListUserUUIDsAfterParams{
			Uuid:  cursor,
			Limit: userBatchSize,
		})
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}

		for _, userID := range batch {
			grouped, err := GroupUser(ctx, g.d, userID)
			if err != nil {
				logg.Error(ctx, "Failed to group tracks for user, skipping.", "userID", userID, "err", err, "grouped", grouped)
				continue
			}
		}

		cursor = batch[len(batch)-1]
	}

	logg.Info(ctx, "Track grouping job finished.")
	return nil
}
