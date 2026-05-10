package astra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures"
)

const (
	// jobTimeout is the maximum time the road closures download job may run.
	jobTimeout = 2 * time.Minute

	// jobKind identifies this job in the job queue and in the inserted_by column.
	jobKind = "roadclosures.astra.downloader"

	// MinRefreshAge is the minimum time between two successful downloads.
	// The job returns early if the most recent insert is younger than this.
	MinRefreshAge = 23 * time.Hour
)

// DownloaderArgs are the arguments for the ASTRA road closures downloader job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return jobKind }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader fetches bike road closures from geo.admin.ch and writes them to the database.
// Each run replaces all previously inserted rows in a single transaction.
// Use [NewDownloader] to create an instance.
type Downloader struct {
	d *db.DB
}

// NewDownloader creates a new [Downloader] instance.
func NewDownloader(d *db.DB) *Downloader {
	return &Downloader{d: d}
}

var _ jobs.Job[DownloaderArgs] = (*Downloader)(nil)

// Run implements [jobs.Job].
// It checks whether the data is stale (older than [MinRefreshAge]), fetches
// road closures from the geo.admin.ch API, and in a single transaction deletes
// all rows previously inserted by this job and inserts the freshly downloaded
// ones together with their H3 cell indices.
func (dl *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	// Skip if data is still fresh.
	lastCreated, err := dl.d.QueryRO().GetLatestRoadClosureCreatedAt(ctx, jobKind)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check last run: %w", err)
	}
	if err == nil && time.Since(lastCreated) < MinRefreshAge {
		logg.Info(ctx, "astra road closures data is recent, skipping download", "lastRun", lastCreated)
		return nil
	}

	logg.Info(ctx, "fetching astra road closures")
	resp, err := Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch road closures: %w", err)
	}
	logg.Info(ctx, "fetched astra road closures", "count", len(resp.Results))

	now := time.Now()

	return dl.d.WithTx(ctx, func(tx *db.Queries) error {
		deleted, err := tx.DeleteRoadClosuresByInsertedBy(ctx, jobKind)
		if err != nil {
			return fmt.Errorf("delete old road closures: %w", err)
		}
		logg.Info(ctx, "deleted old astra road closures", "count", deleted)

		for _, f := range resp.Results {
			if err := insertFeature(ctx, tx, f, now); err != nil {
				return fmt.Errorf("insert feature %d: %w", f.FeatureID, err)
			}
		}

		logg.Info(ctx, "inserted astra road closures", "count", len(resp.Results))
		return nil
	})
}

// insertFeature maps an ASTRA feature into a generic [roadclosures.ClosureInsert]
// and delegates to the shared insert helper.
func insertFeature(ctx context.Context, tx *db.Queries, f Feature, now time.Time) error {
	startsAt, endsAt := parseDurationRange(f.Properties.DurationEn)

	c := roadclosures.ClosureInsert{
		SourceID:        strconv.Itoa(f.FeatureID),
		InsertedBy:      jobKind,
		Type:            f.Properties.SperrungenType,
		StartsAt:        startsAt,
		EndsAt:          endsAt,
		Reason:          roadclosures.NullString(f.Properties.ReasonEn),
		Title:           f.Properties.TitleEn,
		Description:     roadclosures.NullString(f.Properties.AbstractEn),
		ContentProvider: roadclosures.NullString(f.Properties.ContentProviderEn),
		Geometry:        f.Geometry,
		Attribution:     DataAttribution,
	}
	return roadclosures.Insert(ctx, tx, c, now)
}

// parseDurationRange extracts start and end dates from a duration string.
// The expected format is "DD.MM.YYYY – DD.MM.YYYY" (en-dash separated).
// Returns null times when the string does not match this pattern.
func parseDurationRange(s string) (startsAt, endsAt sql.NullTime) {
	// The separator is an en-dash (U+2013), optionally surrounded by spaces.
	parts := strings.SplitN(s, "–", 2)
	if len(parts) != 2 {
		return
	}

	const dateLayout = "02.01.2006"

	start, err := time.Parse(dateLayout, strings.TrimSpace(parts[0]))
	if err != nil {
		return
	}
	end, err := time.Parse(dateLayout, strings.TrimSpace(parts[1]))
	if err != nil {
		return
	}

	startsAt = sql.NullTime{Time: start, Valid: true}
	endsAt = sql.NullTime{Time: end, Valid: true}
	return
}
