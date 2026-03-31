package meteo

import (
	"context"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

type cleanerArgs struct{}

// Kind implements [jobs.Args].
func (cleanerArgs) Kind() string { return "forecast.cleaner" }

var _ jobs.Args = (*cleanerArgs)(nil)

// CleanerArgs returns the args for [Cleaner].
//
//revive:disable:unexported-return
func CleanerArgs() cleanerArgs {
	return cleanerArgs{}
}

// Cleaner removes forecast_files rows (and their associated blobs) whose
// reference_time is older than the most recent forecast run in the database.
// Use [NewCleaner] to create an instance.
type Cleaner struct {
	d *db.DB
}

// NewCleaner creates a new [Cleaner] instance.
func NewCleaner(d *db.DB) *Cleaner {
	return &Cleaner{d: d}
}

var _ jobs.Job[cleanerArgs] = (*Cleaner)(nil)

// Run implements [jobs.Job].
// It deletes all forecast_files rows whose valid_time is in the past.
func (c *Cleaner) Run(ctx context.Context, _ cleanerArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	n, err := c.d.QueryRW().DeleteOutdatedForecastFiles(ctx, time.Now())
	if err != nil {
		return err
	}
	if n > 0 {
		logg.Info(ctx, "cleaned up outdated forecast files", "count", n)
	}
	return nil
}
