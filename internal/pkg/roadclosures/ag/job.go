package ag

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
	// jobTimeout is the maximum time the Aargau downloader may run.
	jobTimeout = 2 * time.Minute

	// jobKind identifies this job in the job queue and in the inserted_by column.
	jobKind = "roadclosures.ag.downloader"

	// MinRefreshAge is the minimum time between two successful downloads.
	// The job returns early if the most recent insert is younger than this.
	MinRefreshAge = 23 * time.Hour

	// contentProvider labels the data source in road_closures.content_provider.
	contentProvider = "Kanton Aargau"
)

// DownloaderArgs are the arguments for the Aargau road closures downloader job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return jobKind }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader fetches construction-site road closures from the Canton of
// Aargau ArcGIS MapServer endpoint and writes them to the database. Each
// run replaces all previously inserted rows in a single transaction.
// Use [NewDownloader] to create an instance.
type Downloader struct {
	d *db.DB
}

// NewDownloader creates a new [Downloader] instance.
func NewDownloader(d *db.DB) *Downloader {
	return &Downloader{d: d}
}

var _ jobs.Job[DownloaderArgs] = (*Downloader)(nil)

// Run implements [jobs.Job]. It refreshes Aargau construction-site closures:
// skips if the most recent insert is within [MinRefreshAge], fetches all
// features, and atomically replaces this job's rows.
//
// The AG feed does not distinguish detours from closures. The closure type
// is derived from the project description and impairment text: features
// containing "sperrung" or "gesperrt" are classified as closed_way; all
// others as detour.
func (dl *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	lastCreated, err := dl.d.QueryRO().GetLatestRoadClosureCreatedAt(ctx, jobKind)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check last run: %w", err)
	}
	if err == nil && time.Since(lastCreated) < MinRefreshAge {
		logg.Info(ctx, "AG road closures data is recent, skipping download", "lastRun", lastCreated)
		return nil
	}

	logg.Info(ctx, "fetching AG road closures")
	resp, err := Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch road closures: %w", err)
	}
	logg.Info(ctx, "fetched AG road closures", "count", len(resp.Features))

	now := time.Now()

	return dl.d.WithTx(ctx, func(tx *db.Queries) error {
		deleted, err := tx.DeleteRoadClosuresByInsertedBy(ctx, jobKind)
		if err != nil {
			return fmt.Errorf("delete old road closures: %w", err)
		}
		logg.Info(ctx, "deleted old AG road closures", "count", deleted)

		var inserted, skipped int
		for _, f := range resp.Features {
			if f.Geometry == nil {
				skipped++
				continue
			}
			if err := insertFeature(ctx, tx, f, now); err != nil {
				return fmt.Errorf("insert feature %d: %w", f.Properties.ObjectID, err)
			}
			inserted++
		}

		logg.Info(ctx, "inserted AG road closures", "count", inserted, "skippedNoGeometry", skipped)
		return nil
	})
}

// insertFeature maps an Aargau feature into a generic [roadclosures.ClosureInsert]
// and delegates to the shared insert helper.
func insertFeature(ctx context.Context, tx *db.Queries, f Feature, now time.Time) error {
	title := featureTitle(f.Properties)

	c := roadclosures.ClosureInsert{
		SourceID:        sourceID(f.Properties.ObjectID),
		InsertedBy:      jobKind,
		Type:            closureTypeFromText(f.Properties.BehinderungKarte, f.Properties.BehinderungTabelle),
		StartsAt:        nullTime(f.Properties.FDate.Time),
		EndsAt:          nullTime(f.Properties.TDate.Time),
		Title:           title,
		Description:     roadclosures.NullString(featureDescription(f.Properties)),
		ContentProvider: roadclosures.NullString(contentProvider),
		Geometry:        f.Geometry,
		Attribution:     DataAttribution,
	}
	return roadclosures.Insert(ctx, tx, c, now)
}

// sourceID returns the canonical SourceID for an AG feature derived from
// its OBJECTID, prefixed with "ag-".
func sourceID(objectID int64) string {
	return "ag-" + strconv.FormatInt(objectID, 10)
}

// featureTitle returns a title combining Gemeinde and Bezeichnung.
func featureTitle(p Properties) string {
	return p.Gemeinde + " - " + p.Bezeichnung
}

// featureDescription returns the BehinderungTabelle text as description.
func featureDescription(p Properties) string {
	return p.BehinderungTabelle
}

// closureTypeFromText derives the closure type from the combined feature
// texts. If the lowercased concatenation contains "sperrung", "gesperrt", or
// "einbahn", the feature is classified as [roadclosures.ClosedWay]; otherwise
// [roadclosures.Detour].
func closureTypeFromText(parts ...string) roadclosures.ClosureType {
	combined := strings.ToLower(strings.Join(parts, " "))
	if strings.Contains(combined, "sperrung") || strings.Contains(combined, "gesperrt") || strings.Contains(combined, "einbahn") {
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
