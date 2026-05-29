package sz

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
	// jobTimeout is the maximum time the Schwyz downloader may run.
	jobTimeout = 2 * time.Minute

	// jobKind identifies this job in the job queue and in the inserted_by column.
	jobKind = "roadclosures.sz.downloader"

	// MinRefreshAge is the minimum time between two successful downloads.
	// The job returns early if the most recent insert is younger than this.
	MinRefreshAge = 23 * time.Hour

	// contentProvider labels the data source in road_closures.content_provider.
	contentProvider = "Kanton Schwyz"
)

// DownloaderArgs are the arguments for the Schwyz road closures downloader job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return jobKind }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader fetches construction-site road closures from the Canton of
// Schwyz WFS endpoint and writes them to the database. Each run replaces
// all previously inserted rows in a single transaction.
// Use [NewDownloader] to create an instance.
type Downloader struct {
	d *db.DB
}

// NewDownloader creates a new [Downloader] instance.
func NewDownloader(d *db.DB) *Downloader {
	return &Downloader{d: d}
}

var _ jobs.Job[DownloaderArgs] = (*Downloader)(nil)

// Run implements [jobs.Job]. It refreshes Schwyz construction-site closures:
// skips if the most recent insert is within [MinRefreshAge], fetches all
// features, and atomically replaces this job's rows.
//
// The SZ feed has no status field, so all features returned by the WFS are
// kept; date fields are best-effort parsed from human-readable strings.
func (dl *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	lastCreated, err := dl.d.QueryRO().GetLatestRoadClosureCreatedAt(ctx, jobKind)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check last run: %w", err)
	}
	if err == nil && time.Since(lastCreated) < MinRefreshAge {
		logg.Info(ctx, "SZ road closures data is recent, skipping download", "lastRun", lastCreated)
		return nil
	}

	logg.Info(ctx, "fetching SZ road closures")
	features, err := Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch road closures: %w", err)
	}
	logg.Info(ctx, "fetched SZ road closures", "count", len(features))

	now := time.Now()

	return dl.d.WithTx(ctx, func(tx *db.Queries) error {
		deleted, err := tx.DeleteRoadClosuresByInsertedBy(ctx, jobKind)
		if err != nil {
			return fmt.Errorf("delete old road closures: %w", err)
		}
		logg.Info(ctx, "deleted old SZ road closures", "count", deleted)

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

		logg.Info(ctx, "inserted SZ road closures", "count", inserted, "skippedNoGeometry", skipped)
		return nil
	})
}

// insertFeature maps a Schwyz feature into a generic [roadclosures.ClosureInsert]
// and delegates to the shared insert helper.
//
// The SZ feed does not carry a structured "type" field, so every closure is
// reported as a plain way closure. The Beschreibung and Behinderungsbemerkung
// fields are concatenated into the Description so end users see both the
// reason and the resulting traffic management.
func insertFeature(ctx context.Context, tx *db.Queries, f Feature, now time.Time) error {
	title := f.Lokalname
	if title == "" {
		title = contentProvider
	}

	c := roadclosures.ClosureInsert{
		SourceID:        f.SourceID,
		InsertedBy:      jobKind,
		Type:            roadclosures.ClosedWay,
		StartsAt:        nullTime(f.Baubeginn),
		EndsAt:          nullTime(f.Inbetriebnahme),
		Reason:          roadclosures.NullString(f.Beschreibung),
		Title:           title,
		Description:     roadclosures.NullString(buildDescription(f)),
		ContentProvider: roadclosures.NullString(contentProvider),
		Geometry:        f.Geometry,
		Attribution:     DataAttribution,
	}
	return roadclosures.Insert(ctx, tx, c, now)
}

// buildDescription concatenates the traffic-management note and the optional
// project link into the description column. Returns an empty string when
// both inputs are empty.
func buildDescription(f Feature) string {
	parts := make([]string, 0, 2)
	if s := strings.TrimSpace(f.Behinderungsbemerkung); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(f.Link); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n\n")
}

// nullTime wraps a time.Time as sql.NullTime, treating the zero time as NULL.
func nullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}
