package forecast

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

// targetDownloadVariables is the list of forecast variables stored by the downloader job.
var targetDownloadVariables = []string{"U_10M", "V_10M", "TOT_PREC"}

// DownloaderArgs are the arguments for the forecast downloader job.
// No configuration fields are needed; behaviour is fixed at registration time.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return "forecast.downloader" }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader implements the forecast download job.
// Use [NewDownloader] to create an instance.
type Downloader struct {
	d *db.DB
}

// NewDownloader creates a new [Downloader] instance.
func NewDownloader(d *db.DB) *Downloader {
	return &Downloader{d: d}
}

var _ jobs.Job[DownloaderArgs] = (*Downloader)(nil)

// pendingFile holds metadata and the local path for a downloaded GRIB2 file
// that is waiting to be committed to the database.
type pendingFile struct {
	path          string
	referenceTime time.Time
	validTime     time.Time
	variable      string
	boundsMinLat  sql.NullFloat64
	boundsMinLon  sql.NullFloat64
	boundsMaxLat  sql.NullFloat64
	boundsMaxLon  sql.NullFloat64
}

// Run implements [jobs.Job].
// It queries the STAC API for the most recent forecast run and, if it is newer
// than what is already stored in the database, downloads all non-perturbed
// files for the target variables (all available horizons) to a temporary
// directory and commits them atomically to the blobs and forecast_files tables.
func (d *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	items, refTime, err := fetchItemsForVariables(ctx, targetDownloadVariables)
	if err != nil {
		return fmt.Errorf("fetch STAC items: %w", err)
	}
	if len(items) == 0 {
		logg.Info(ctx, "No forecast items found for target variables")
		return nil
	}

	logg.Info(ctx, "Latest forecast run available", "referenceTime", refTime, "itemCount", len(items))

	// Skip download when this reference time is already stored.
	existing, err := d.d.QueryRO().GetLatestForecastReferenceTime(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("query latest forecast reference time: %w", err)
	}
	if err == nil && !existing.Before(refTime) {
		logg.Info(ctx, "Forecast data is already up to date", "referenceTime", refTime)
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "forecast-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
			logg.Error(ctx, "Failed to remove forecast temp dir", "path", tmpDir, "err", removeErr)
		}
	}()

	// Download only non-perturbed files, covering all available horizons.
	var files []pendingFile
	for i, item := range items {
		if item.Properties.Perturbed {
			continue
		}
		if item.Properties.Horizon == "" || item.Properties.Variable == "" {
			logg.Warn(ctx, "Skipping item with missing metadata", "id", item.ID)
			continue
		}

		horizon, parseErr := parseISO8601Duration(item.Properties.Horizon)
		if parseErr != nil {
			return fmt.Errorf("parse horizon %q for item %s: %w", item.Properties.Horizon, item.ID, parseErr)
		}

		var assetURL string
		for _, asset := range item.Assets {
			assetURL = asset.Href
			break
		}
		if assetURL == "" {
			logg.Warn(ctx, "Skipping item with no assets", "id", item.ID)
			continue
		}

		localPath := filepath.Join(tmpDir, fmt.Sprintf("%04d.grib2", i))
		logg.Debug(ctx, "Downloading forecast file",
			"variable", item.Properties.Variable,
			"horizon", item.Properties.Horizon)
		if downloadErr := downloadFile(ctx, assetURL, localPath); downloadErr != nil {
			return fmt.Errorf("download %s/%s: %w",
				item.Properties.Variable, item.Properties.Horizon, downloadErr)
		}

		pf := pendingFile{
			path:          localPath,
			referenceTime: refTime,
			validTime:     refTime.Add(horizon),
			variable:      item.Properties.Variable,
		}
		nullBBoxFromItem(item, &pf)
		files = append(files, pf)
	}

	if len(files) == 0 {
		logg.Info(ctx, "No non-perturbed forecast files to store", "referenceTime", refTime)
		return nil
	}

	logg.Info(ctx, "Storing forecast data",
		"referenceTime", refTime,
		"fileCount", len(files),
		"tmpDir", tmpDir)

	// Atomically write all blobs and forecast_file records in a single transaction.
	err = d.d.WithTx(ctx, func(tx *db.Queries) error {
		for _, f := range files {
			content, readErr := os.ReadFile(f.path)
			if readErr != nil {
				return fmt.Errorf("read %s: %w", f.path, readErr)
			}

			b, blobErr := blob.Create(ctx, tx, content, blob.CompressionNone)
			if blobErr != nil {
				return fmt.Errorf("create blob for %s: %w", f.path, blobErr)
			}

			if _, dbErr := tx.CreateForecastFile(ctx, db.CreateForecastFileParams{
				CreatedAt:     time.Now(),
				ReferenceTime: f.referenceTime,
				ValidTime:     f.validTime,
				Variable:      f.variable,
				BoundsMinLat:  f.boundsMinLat,
				BoundsMinLon:  f.boundsMinLon,
				BoundsMaxLat:  f.boundsMaxLat,
				BoundsMaxLon:  f.boundsMaxLon,
				BlobID:        b.ID,
			}); dbErr != nil {
				return fmt.Errorf("create forecast_file record for %s validTime=%s: %w",
					f.variable, f.validTime.Format(time.RFC3339), dbErr)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("write forecast to database: %w", err)
	}

	logg.Info(ctx, "Stored new forecast data", "referenceTime", refTime, "fileCount", len(files))
	return nil
}

// nullBBoxFromItem converts the STAC item BBox ([min_lon, min_lat, max_lon, max_lat])
// into nullable float64 values and assigns them to pf.
func nullBBoxFromItem(item stacItem, pf *pendingFile) {
	if len(item.BBox) == 4 &&
		!math.IsNaN(item.BBox[0]) && !math.IsNaN(item.BBox[1]) &&
		!math.IsNaN(item.BBox[2]) && !math.IsNaN(item.BBox[3]) {
		pf.boundsMinLon = sql.NullFloat64{Float64: item.BBox[0], Valid: true}
		pf.boundsMinLat = sql.NullFloat64{Float64: item.BBox[1], Valid: true}
		pf.boundsMaxLon = sql.NullFloat64{Float64: item.BBox[2], Valid: true}
		pf.boundsMaxLat = sql.NullFloat64{Float64: item.BBox[3], Valid: true}
	}
}
