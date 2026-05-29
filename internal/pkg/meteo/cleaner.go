package meteo

import (
	"context"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db/forecastdb"
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

// emptyForecastMaxAge is the minimum age a forecast row must reach before it
// is eligible for deletion when it has no associated forecast_files.
const emptyForecastMaxAge = 24 * time.Hour

// pastFileRetention is how far past now a forecast_files row must fall before
// it is eligible for deletion. The buffer keeps the immediate same-run
// predecessor that [forecast.Load] needs in order to de-average ICON-CH1-EPS
// running-mean and running-accumulation variables for queries near the
// current hour. A file at valid_time = V is the predecessor for queries with
// start in [V+1h, V+2h); assuming the frontend lower-bounds start at the
// current time, files with V >= now - 2h may still be needed.
const pastFileRetention = 2 * time.Hour

// Cleaner removes forecast_files rows whose valid_time fell more than
// [pastFileRetention] in the past, and any forecasts rows that no longer
// have associated files and are older than [emptyForecastMaxAge].
// Use [NewCleaner] to create an instance.
type Cleaner struct {
	d *forecastdb.DB
}

// NewCleaner creates a new [Cleaner] instance.
func NewCleaner(d *forecastdb.DB) *Cleaner {
	return &Cleaner{d: d}
}

var _ jobs.Job[cleanerArgs] = (*Cleaner)(nil)

// Run implements [jobs.Job].
// It deletes all forecast_files rows whose valid_time is older than
// now - [pastFileRetention], then deletes any forecasts rows that have no
// remaining files and whose reference_time is older than
// [emptyForecastMaxAge].
func (c *Cleaner) Run(ctx context.Context, _ cleanerArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	now := time.Now()

	nFiles, err := c.d.QueryRW().DeleteOutdatedForecastFiles(ctx, now.Add(-pastFileRetention))
	if err != nil {
		return err
	}
	if nFiles > 0 {
		logg.Info(ctx, "cleaned up outdated forecast files", "count", nFiles)
	}

	nForecasts, err := c.d.QueryRW().DeleteEmptyForecastsOlderThan(ctx, now.Add(-emptyForecastMaxAge))
	if err != nil {
		return err
	}
	if nForecasts > 0 {
		logg.Info(ctx, "cleaned up empty old forecasts", "count", nForecasts)
	}

	return nil
}
