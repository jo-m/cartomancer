package app

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

// DemoTrackPurgePeriod is the interval at which all tracks are deleted in demo mode.
const DemoTrackPurgePeriod = 30 * time.Minute

// demoTriggers are the SQL statements to install demo mode database triggers.
// They prevent modifications to the users and email_verifications tables,
// while still allowing updates to last_login_at, last_active_at (activity tracking)
// and session-related operations.
var demoTriggers = []string{
	// Block INSERT on users.
	`CREATE TEMPORARY TRIGGER demo_users_no_insert
	 BEFORE INSERT ON users
	 BEGIN
	     SELECT RAISE(ABORT, 'demo mode: user creation is disabled');
	 END`,

	// Block DELETE on users.
	`CREATE TEMPORARY TRIGGER demo_users_no_delete
	 BEFORE DELETE ON users
	 BEGIN
	     SELECT RAISE(ABORT, 'demo mode: user deletion is disabled');
	 END`,

	// Block UPDATE on users, except for last_login_at and last_active_at.
	`CREATE TEMPORARY TRIGGER demo_users_no_update
	 BEFORE UPDATE ON users
	 WHEN (
	     OLD.uuid            IS NOT NEW.uuid
	  OR OLD.created_at      IS NOT NEW.created_at
	  OR OLD.updated_at      IS NOT NEW.updated_at
	  OR OLD.email           IS NOT NEW.email
	  OR OLD.name            IS NOT NEW.name
	  OR OLD.password_hash   IS NOT NEW.password_hash
	  OR OLD.otp_secret      IS NOT NEW.otp_secret
	  OR OLD.admin           IS NOT NEW.admin
	  OR OLD.email_confirmed    IS NOT NEW.email_confirmed
	  OR OLD.avatar_seed        IS NOT NEW.avatar_seed
	  OR OLD.location_name IS NOT NEW.location_name
	  OR OLD.location_lat  IS NOT NEW.location_lat
	  OR OLD.location_lon  IS NOT NEW.location_lon
	 )
	 BEGIN
	     SELECT RAISE(ABORT, 'demo mode: user modification is disabled');
	 END`,

	// Block all writes on email_verifications.
	`CREATE TEMPORARY TRIGGER demo_email_verifications_no_insert
	 BEFORE INSERT ON email_verifications
	 BEGIN
	     SELECT RAISE(ABORT, 'demo mode: email verification is disabled');
	 END`,

	`CREATE TEMPORARY TRIGGER demo_email_verifications_no_update
	 BEFORE UPDATE ON email_verifications
	 BEGIN
	     SELECT RAISE(ABORT, 'demo mode: email verification is disabled');
	 END`,

	`CREATE TEMPORARY TRIGGER demo_email_verifications_no_delete
	 BEFORE DELETE ON email_verifications
	 BEGIN
	     SELECT RAISE(ABORT, 'demo mode: email verification is disabled');
	 END`,
}

// InstallDemoTriggers creates temporary database triggers that lock down
// the users and email_verifications tables for demo mode.
// Temporary triggers live only for the lifetime of the database connection.
// Only needs to be called on the RW connection, since RO connections cannot write.
func InstallDemoTriggers(ctx context.Context, conn *sql.DB) error {
	for _, stmt := range demoTriggers {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to install demo trigger: %w", err)
		}
	}
	logg.Info(ctx, "installed demo mode database triggers")
	return nil
}

const demoTrackPurgeJobKind = "demo.track_purge"

// DemoTrackPurgeArgs are the arguments for the demo track purge job.
type DemoTrackPurgeArgs struct{}

var _ jobs.Args = (*DemoTrackPurgeArgs)(nil)

// Kind implements [jobs.Args].
func (a DemoTrackPurgeArgs) Kind() string { return demoTrackPurgeJobKind }

// DemoTrackPurger is a job that deletes all tracks and their associated blobs.
// Use [NewDemoTrackPurger] to create a new instance.
type DemoTrackPurger struct {
	d *db.DB
}

var _ jobs.Job[DemoTrackPurgeArgs] = (*DemoTrackPurger)(nil)

// NewDemoTrackPurger creates a new [DemoTrackPurger] instance.
func NewDemoTrackPurger(d *db.DB) *DemoTrackPurger {
	return &DemoTrackPurger{d: d}
}

// Run implements [jobs.Job].
func (p *DemoTrackPurger) Run(ctx context.Context, _ DemoTrackPurgeArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	n, err := p.d.QueryRW().DeleteAllTracks(ctx)
	if err != nil {
		return fmt.Errorf("failed to purge tracks: %w", err)
	}
	if n > 0 {
		logg.Info(ctx, "demo mode: purged all tracks", "count", n)
	}
	return nil
}
