package ktzh

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/streetlights"
)

const (
	// jobTimeout is the maximum time the canton downloader may run. Roughly
	// 23k features are fetched over about 23 paginated requests.
	jobTimeout = 10 * time.Minute

	// jobKind identifies this job in the job queue and in the inserted_by column.
	jobKind = "streetlights.ktzh.downloader"

	// MinRefreshAge is the minimum time between two successful downloads.
	// Street lighting changes slowly, so monthly is enough. The job returns
	// early if the most recent insert is younger than this.
	MinRefreshAge = 30 * 24 * time.Hour
)

// DownloaderArgs are the arguments for the canton street lighting downloader
// job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return jobKind }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader fetches street lighting operated by the Canton of Zurich from
// the OGD WFS endpoint and writes it to the database. Each run replaces all
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

// Run implements [jobs.Job]. It refreshes the canton's street lighting:
// skips if the most recent insert is within [MinRefreshAge], fetches all
// features, and atomically replaces this job's rows.
func (dl *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	lastCreated, err := dl.d.QueryRO().GetLatestStreetlightCreatedAt(ctx, jobKind)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check last run: %w", err)
	}
	if err == nil && time.Since(lastCreated) < MinRefreshAge {
		logg.Info(ctx, "ktzh streetlights data is recent, skipping download", "lastRun", lastCreated)
		return nil
	}

	logg.Info(ctx, "fetching ktzh streetlights")
	features, err := Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch streetlights: %w", err)
	}
	logg.Info(ctx, "fetched ktzh streetlights", "count", len(features))

	now := time.Now()

	return dl.d.WithTx(ctx, func(tx *db.Queries) error {
		deleted, err := tx.DeleteStreetlightsByInsertedBy(ctx, jobKind)
		if err != nil {
			return fmt.Errorf("delete old streetlights: %w", err)
		}
		logg.Info(ctx, "deleted old ktzh streetlights", "count", deleted)

		var inserted, skipped int
		for _, f := range features {
			err := insertFeature(ctx, tx, f, now)
			if errors.Is(err, streetlights.ErrNilGeometry) || errors.Is(err, streetlights.ErrNonPointGeometry) {
				logg.Debug(ctx, "skipping streetlight without point geometry", "sourceId", f.SourceID, "err", err)
				skipped++
				continue
			}
			if err != nil {
				return fmt.Errorf("insert feature %s: %w", f.SourceID, err)
			}
			inserted++
		}

		logg.Info(ctx, "inserted ktzh streetlights", "count", inserted, "skipped", skipped)
		return nil
	})
}

// insertFeature maps a canton feature into a generic
// [streetlights.StreetlightInsert] and delegates to the shared insert helper.
func insertFeature(ctx context.Context, tx *db.Queries, f Feature, now time.Time) error {
	s := streetlights.StreetlightInsert{
		SourceID:    f.SourceID,
		InsertedBy:  jobKind,
		Geometry:    f.Geometry,
		Attribution: DataAttribution,
	}
	return streetlights.Insert(ctx, tx, s, now)
}
