package zh

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
	// jobTimeout is the maximum time the Zurich downloader may run.
	jobTimeout = 2 * time.Minute

	// jobKind identifies this job in the job queue and in the inserted_by column.
	jobKind = "roadclosures.zh.downloader"

	// MinRefreshAge is the minimum time between two successful downloads.
	// The job returns early if the most recent insert is younger than this.
	MinRefreshAge = 23 * time.Hour

	// contentProvider labels the data source in road_closures.content_provider.
	contentProvider = "Tiefbauamt Kanton Zürich"
)

// DownloaderArgs are the arguments for the Zurich road closures downloader job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return jobKind }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader fetches construction-site road closures from the Canton of Zurich
// WFS endpoint and writes them to the database. Each run replaces all
// previously inserted rows in a single transaction.
// Use [NewDownloader] to create an instance.
type Downloader struct {
	d *db.DB
}

// NewDownloader creates a new [Downloader] instance.
func NewDownloader(d *db.DB) *Downloader {
	return &Downloader{d: d}
}

var _ jobs.Job[DownloaderArgs] = (*Downloader)(nil)

// Run implements [jobs.Job]. It refreshes Zurich construction-site
// closures: skips if the most recent insert is within [MinRefreshAge],
// fetches all features, filters by construction status, and atomically
// replaces this job's rows.
func (dl *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	lastCreated, err := dl.d.QueryRO().GetLatestRoadClosureCreatedAt(ctx, jobKind)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check last run: %w", err)
	}
	if err == nil && time.Since(lastCreated) < MinRefreshAge {
		logg.Info(ctx, "zh road closures data is recent, skipping download", "lastRun", lastCreated)
		return nil
	}

	logg.Info(ctx, "fetching zh road closures")
	features, err := Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch road closures: %w", err)
	}
	logg.Info(ctx, "fetched zh road closures", "count", len(features))

	kept := filterActive(features)
	logg.Info(ctx, "kept active zh road closures", "count", len(kept), "dropped", len(features)-len(kept))

	now := time.Now()

	return dl.d.WithTx(ctx, func(tx *db.Queries) error {
		deleted, err := tx.DeleteRoadClosuresByInsertedBy(ctx, jobKind)
		if err != nil {
			return fmt.Errorf("delete old road closures: %w", err)
		}
		logg.Info(ctx, "deleted old zh road closures", "count", deleted)

		var inserted, skipped int
		for _, f := range kept {
			if f.Geometry == nil {
				skipped++
				continue
			}
			if err := insertFeature(ctx, tx, f, now); err != nil {
				return fmt.Errorf("insert feature %s: %w", f.GMLID, err)
			}
			inserted++
		}

		logg.Info(ctx, "inserted zh road closures", "count", inserted, "skippedNoGeometry", skipped)
		return nil
	})
}

// filterActive returns only features whose status_baustelle indicates the
// site is active or upcoming. Past sites are skipped.
func filterActive(in []Feature) []Feature {
	out := make([]Feature, 0, len(in))
	for _, f := range in {
		if isActiveStatus(f.StatusBaustelle) {
			out = append(out, f)
		}
	}
	return out
}

// isActiveStatus reports whether a status_baustelle value indicates either an
// in-progress or upcoming construction site.
func isActiveStatus(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(s, "aktiv") || strings.HasPrefix(s, "zukünftig")
}

// insertFeature maps a Zurich feature into a generic [roadclosures.ClosureInsert]
// and delegates to the shared insert helper.
func insertFeature(ctx context.Context, tx *db.Queries, f Feature, now time.Time) error {
	title := f.Strassenname
	if title == "" {
		title = f.Gemeindename
	}

	c := roadclosures.ClosureInsert{
		SourceID:        f.GMLID,
		InsertedBy:      jobKind,
		Type:            "closed_way",
		StartsAt:        nullTime(f.DatumBaubeginn),
		EndsAt:          nullTime(f.DatumBauende),
		Reason:          roadclosures.NullString(f.Beschreibung),
		Title:           title,
		Description:     roadclosures.NullString(f.Verkehrsfuehrung),
		ContentProvider: roadclosures.NullString(contentProvider),
		Geometry:        f.Geometry,
		Attribution:     DataAttribution,
	}
	return roadclosures.Insert(ctx, tx, c, now)
}

// nullTime wraps a time.Time as sql.NullTime, treating the zero time as NULL.
func nullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}
