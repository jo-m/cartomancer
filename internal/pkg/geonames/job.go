package geonames

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

const (
	// downloaderTimeout is the maximum time the geonames download job may run.
	downloaderTimeout = 30 * time.Minute

	// minImportAge is the minimum age of the last import before a new one is attempted.
	minImportAge = 6 * 24 * time.Hour
)

// DownloaderArgs are the arguments for the GeoNames downloader job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return "geonames.downloader" }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader downloads the GeoNames allCountries dataset and imports it.
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
// It checks whether a recent import exists. If the last import is younger
// than [minImportAge], it returns early. Otherwise it downloads and imports
// the full GeoNames dataset.
func (d *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, downloaderTimeout)
	defer cancel()

	// Check if a recent import exists.
	latest, err := d.d.QueryRO().GetLatestGeonameImport(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check latest import: %w", err)
	}
	if err == nil && time.Since(latest.CreatedAt) < minImportAge {
		logg.Info(ctx, "geonames data is recent, skipping download", "lastImport", latest.CreatedAt)
		return nil
	}

	logg.Info(ctx, "downloading GeoNames allCountries.zip")
	zipPath, err := DownloadAllCountries(ctx)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(zipPath); removeErr != nil {
			logg.Error(ctx, "failed to remove geonames temp file", "path", zipPath, "err", removeErr)
		}
	}()

	logg.Info(ctx, "importing GeoNames data")
	rowCount, err := ImportAllCountries(ctx, d.d, zipPath)
	if err != nil {
		return fmt.Errorf("import: %w", err)
	}

	logg.Info(ctx, "downloading GeoNames admin codes")
	admin1Data, admin2Data, err := DownloadAdminCodes(ctx)
	if err != nil {
		return fmt.Errorf("download admin codes: %w", err)
	}

	if _, err := ImportAdmin1Codes(ctx, d.d, bytes.NewReader(admin1Data)); err != nil {
		return fmt.Errorf("import admin1 codes: %w", err)
	}
	if _, err := ImportAdmin2Codes(ctx, d.d, bytes.NewReader(admin2Data)); err != nil {
		return fmt.Errorf("import admin2 codes: %w", err)
	}

	// Record the import.
	_, err = d.d.QueryRW().CreateGeonameImport(ctx, db.CreateGeonameImportParams{
		CreatedAt: time.Now(),
		RowCount:  int64(rowCount),
	})
	if err != nil {
		return fmt.Errorf("record import: %w", err)
	}

	// Clean up old import records.
	_, _ = d.d.QueryRW().DeleteOldGeonameImports(ctx)

	logg.Info(ctx, "geonames import complete", "rows", rowCount)
	return nil
}
