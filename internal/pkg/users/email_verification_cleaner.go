// Package users deals with users.
package users

import (
	"context"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

type emailVerificationCleanerArgs struct{}

// Kind implements [jobs.Args].
func (a emailVerificationCleanerArgs) Kind() string { return "users.email-verification-cleaner" }

var _ jobs.Args = (*emailVerificationCleanerArgs)(nil)

// EmailVerificationCleanerArgs returns the args for [EmailVerificationCleaner].
//
//revive:disable:unexported-return
func EmailVerificationCleanerArgs() emailVerificationCleanerArgs {
	return emailVerificationCleanerArgs{}
}

// EmailVerificationCleaner removes expired email verification records.
// Use [NewEmailVerificationCleaner] to create a new instance.
type EmailVerificationCleaner struct {
	d *db.DB
}

// NewEmailVerificationCleaner creates a new [EmailVerificationCleaner] instance.
func NewEmailVerificationCleaner(d *db.DB) *EmailVerificationCleaner {
	return &EmailVerificationCleaner{d: d}
}

var _ jobs.Job[emailVerificationCleanerArgs] = (*EmailVerificationCleaner)(nil)

// Run implements [jobs.Job].
// It deletes expired email verification records, then removes user accounts
// that never confirmed their email and no longer have a pending verification.
// For users who changed their email but did not confirm the new one, the expired
// verification is simply removed; they remain on their previously confirmed email.
// Both steps run in a single transaction to avoid a window where an unconfirmed
// user exists without a verification but has not yet been deleted.
func (c *EmailVerificationCleaner) Run(ctx context.Context, _ emailVerificationCleanerArgs) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var n, m int64
	err := c.d.WithTx(ctx, func(q *db.Queries) error {
		var txErr error
		n, txErr = q.DeleteExpiredEmailVerifications(ctx, time.Now())
		if txErr != nil {
			return txErr
		}
		m, txErr = q.DeleteUnconfirmedUsersWithoutVerification(ctx)
		return txErr
	})
	if err != nil {
		return err
	}

	if n > 0 {
		logg.Info(ctx, "cleaned up expired email verifications", "count", n)
	}
	if m > 0 {
		logg.Info(ctx, "deleted unconfirmed user accounts", "count", m)
	}

	return nil
}
