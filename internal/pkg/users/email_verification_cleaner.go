package users

import (
	"context"
	"time"

	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
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
func (c *EmailVerificationCleaner) Run(ctx context.Context, _ emailVerificationCleanerArgs) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	n, err := c.d.QueryRW().DeleteExpiredEmailVerifications(ctx, time.Now())
	if n > 0 {
		logg.Info(ctx, "cleaned up expired email verifications", "count", n)
	}
	return err
}
