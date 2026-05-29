package sg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures"
)

const (
	// jobTimeout is the maximum time the St. Gallen downloader may run.
	jobTimeout = 2 * time.Minute

	// jobKind identifies this job in the job queue and in the inserted_by column.
	jobKind = "roadclosures.sg.downloader"

	// MinRefreshAge is the minimum time between two successful downloads.
	// The job returns early if the most recent insert is younger than this.
	MinRefreshAge = 23 * time.Hour

	// contentProvider labels the data source in road_closures.content_provider.
	contentProvider = "Kanton St. Gallen"
)

// DownloaderArgs are the arguments for the St. Gallen road closures downloader job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return jobKind }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader fetches construction-site road closures from the Canton of
// St. Gallen open data WFS endpoint and writes them to the database.
// Each run replaces all previously inserted rows in a single transaction.
type Downloader struct {
	d *db.DB
}

// NewDownloader creates a new [Downloader] instance.
func NewDownloader(d *db.DB) *Downloader {
	return &Downloader{d: d}
}

var _ jobs.Job[DownloaderArgs] = (*Downloader)(nil)

// Run implements [jobs.Job]. It refreshes St. Gallen construction-site
// closures: skips if the most recent insert is within [MinRefreshAge],
// fetches all features, and atomically replaces this job's rows.
//
// The SG feed has no status field, so all features returned by the WFS are
// inserted; the shared DB query filters by ends_at when serving the API.
// Type is derived from the feature text: features containing "sperrung" or
// "gesperrt" are classified as closed_way; all others as detour.
func (dl *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	lastCreated, err := dl.d.QueryRO().GetLatestRoadClosureCreatedAt(ctx, jobKind)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check last run: %w", err)
	}
	if err == nil && time.Since(lastCreated) < MinRefreshAge {
		logg.Info(ctx, "SG road closures data is recent, skipping download", "lastRun", lastCreated)
		return nil
	}

	logg.Info(ctx, "fetching SG road closures")
	features, err := Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch road closures: %w", err)
	}
	logg.Info(ctx, "fetched SG road closures", "count", len(features))

	now := time.Now()

	return dl.d.WithTx(ctx, func(tx *db.Queries) error {
		deleted, err := tx.DeleteRoadClosuresByInsertedBy(ctx, jobKind)
		if err != nil {
			return fmt.Errorf("delete old road closures: %w", err)
		}
		logg.Info(ctx, "deleted old SG road closures", "count", deleted)

		var inserted, skipped int
		for _, f := range features {
			if f.Geometry == nil {
				skipped++
				continue
			}
			if err := insertFeature(ctx, tx, f, now); err != nil {
				return fmt.Errorf("insert feature %s: %w", f.SourceID, err)
			}
			inserted++
		}

		logg.Info(ctx, "inserted SG road closures", "count", inserted, "skippedNoGeometry", skipped)
		return nil
	})
}

// insertFeature maps a St. Gallen feature into a generic [roadclosures.ClosureInsert]
// and delegates to the shared insert helper.
func insertFeature(ctx context.Context, tx *db.Queries, f Feature, now time.Time) error {
	title := f.Bew
	if title == "" {
		title = contentProvider
	}

	c := roadclosures.ClosureInsert{
		SourceID:        f.SourceID,
		InsertedBy:      jobKind,
		Type:            closureTypeFromText(title, f.Adresse),
		StartsAt:        nullTime(f.Beginn),
		EndsAt:          nullTime(f.Ende),
		Title:           title,
		Description:     roadclosures.NullString(f.Adresse),
		ContentProvider: roadclosures.NullString(contentProvider),
		Geometry:        f.Geometry,
		Attribution:     DataAttribution,
	}
	return roadclosures.Insert(ctx, tx, c, now)
}

// closureTypeFromText derives the closure type from the combined title and
// description text. If the lowercased concatenation contains "sperrung" or
// "gesperrt", the feature is classified as [roadclosures.ClosedWay]; otherwise
// [roadclosures.Detour].
func closureTypeFromText(title, description string) roadclosures.ClosureType {
	combined := strings.ToLower(title + " " + description)
	if strings.Contains(combined, "sperrung") || strings.Contains(combined, "gesperrt") {
		return roadclosures.ClosedWay
	}
	return roadclosures.Detour
}

// nullTime wraps a time.Time as sql.NullTime, treating the zero time as NULL.
func nullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}
