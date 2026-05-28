package tg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures"
)

const (
	// jobTimeout is the maximum time the Thurgau downloader may run.
	jobTimeout = 2 * time.Minute

	// jobKind identifies this job in the job queue and in the inserted_by column.
	jobKind = "roadclosures.tg.downloader"

	// MinRefreshAge is the minimum time between two successful downloads.
	// The job returns early if the most recent insert is younger than this.
	MinRefreshAge = 23 * time.Hour

	// contentProvider labels the data source in road_closures.content_provider.
	contentProvider = "Kanton Thurgau"

	// dateLayout matches the YYYY-MM-DD strings used by the ThurGIS feed
	// for terminvon and terminbis.
	dateLayout = "2006-01-02"

	// closureType is the road_closures.type assigned to every TG feature.
	// The ThurGIS schema has no field distinguishing detours from closures,
	// so we default to closed_way until upstream provides a usable signal.
	closureType = "closed_way"
)

// DownloaderArgs are the arguments for the Thurgau road closures downloader job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return jobKind }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader fetches construction-site road closures from the ThurGIS
// portal and writes them to the database. Each run replaces all previously
// inserted rows in a single transaction. Use [NewDownloader] to create an
// instance.
type Downloader struct {
	d *db.DB
}

// NewDownloader creates a new [Downloader] instance.
func NewDownloader(d *db.DB) *Downloader {
	return &Downloader{d: d}
}

var _ jobs.Job[DownloaderArgs] = (*Downloader)(nil)

// Run implements [jobs.Job]. It refreshes Thurgau construction-site
// closures: skips if the most recent insert is within [MinRefreshAge],
// fetches all features (already reprojected to WGS84 by [Fetch]), and
// atomically replaces this job's rows.
func (dl *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	lastCreated, err := dl.d.QueryRO().GetLatestRoadClosureCreatedAt(ctx, jobKind)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check last run: %w", err)
	}
	if err == nil && time.Since(lastCreated) < MinRefreshAge {
		logg.Info(ctx, "TG road closures data is recent, skipping download", "lastRun", lastCreated)
		return nil
	}

	logg.Info(ctx, "fetching TG road closures")
	resp, err := Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch road closures: %w", err)
	}
	logg.Info(ctx, "fetched TG road closures", "count", len(resp.Results))

	now := time.Now()

	return dl.d.WithTx(ctx, func(tx *db.Queries) error {
		deleted, err := tx.DeleteRoadClosuresByInsertedBy(ctx, jobKind)
		if err != nil {
			return fmt.Errorf("delete old road closures: %w", err)
		}
		logg.Info(ctx, "deleted old TG road closures", "count", deleted)

		var inserted, skipped int
		for _, f := range resp.Results {
			if f.Geometry == nil {
				skipped++
				continue
			}
			if err := insertFeature(ctx, tx, f, now); err != nil {
				return fmt.Errorf("insert feature %s: %w", f.Properties.ObjectID, err)
			}
			inserted++
		}

		logg.Info(ctx, "inserted TG road closures", "count", inserted, "skippedNoGeometry", skipped)
		return nil
	})
}

// insertFeature maps a ThurGIS feature into a generic [roadclosures.ClosureInsert]
// and delegates to the shared insert helper.
func insertFeature(ctx context.Context, tx *db.Queries, f Feature, now time.Time) error {
	c := roadclosures.ClosureInsert{
		SourceID:        sourceID(f.Properties.ObjectID),
		InsertedBy:      jobKind,
		Type:            closureType,
		StartsAt:        parseDate(f.Properties.TerminVon),
		EndsAt:          parseDate(f.Properties.TerminBis),
		Title:           featureTitle(f.Properties),
		Description:     roadclosures.NullString(f.Properties.Taetigkeitsbeschrieb),
		ContentProvider: roadclosures.NullString(contentProvider),
		Geometry:        f.Geometry,
		Attribution:     DataAttribution,
	}
	return roadclosures.Insert(ctx, tx, c, now)
}

// sourceID returns the canonical SourceID for a ThurGIS feature derived
// from its upstream objectid, prefixed with "tg-". An empty objectid yields
// "tg-" so that callers can still detect missing IDs at write time.
func sourceID(objectID string) string {
	return "tg-" + objectID
}

// featureTitle returns the project name, falling back to the project number
// when the name is empty so that the row is never inserted with a blank title.
func featureTitle(p Properties) string {
	if p.Projektbezeichnung != "" {
		return p.Projektbezeichnung
	}
	return p.Projektnummer
}

// parseDate parses a YYYY-MM-DD date string from the feed. An empty string
// or an unparseable value yields an invalid sql.NullTime, which the shared
// insert layer stores as SQL NULL.
func parseDate(s string) sql.NullTime {
	if s == "" {
		return sql.NullTime{}
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}
