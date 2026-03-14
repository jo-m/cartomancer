package meteo

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/meteo/vars"
)

// DownloadVariables is the list of forecast variables stored by the downloader job.
var DownloadVariables = []vars.Variable{vars.VarU10m, vars.VarV10m, vars.VarTotPr, vars.VarT2m}

// DownloaderArgs are the arguments for the forecast downloader job.
// No configuration fields are needed; behaviour is fixed at registration time.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return "meteo.downloader" }

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

// meteoFileKey is the natural unique key for a forecast_files row.
type meteoFileKey struct {
	variable  string
	validTime time.Time
}

// Run implements [jobs.Job].
// It fetches the manifest for the newest STAC forecast run, determines which
// files are not yet stored in the database, downloads only those files to a
// temporary directory, and then commits them atomically to the blobs and
// forecast_files tables.
func (d *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Stage 1: fetch file manifest from STAC (no downloads yet).
	manifest, err := GetNewestForecast(ctx, DownloadVariables, NoHorizonLimit, false)
	if err != nil {
		return fmt.Errorf("fetch forecast manifest: %w", err)
	}

	// Stage 2: determine which files are not yet in the database (read-only, no lock held).
	existingRows, err := d.d.QueryRO().ListForecastFileKeysForReferenceTime(ctx, manifest.ReferenceTime)
	if err != nil {
		return fmt.Errorf("query existing forecast files: %w", err)
	}
	existing := make(map[meteoFileKey]struct{}, len(existingRows))
	for _, row := range existingRows {
		existing[meteoFileKey{row.Variable, row.ValidTime}] = struct{}{}
	}

	gridExists, err := d.d.QueryRO().ForecastGridExistsForReferenceTime(ctx, manifest.ReferenceTime)
	if err != nil {
		return fmt.Errorf("check existing grid constants: %w", err)
	}
	needGrid := gridExists == 0

	var newFiles []ForecastFile
	for _, f := range manifest.Files {
		if _, ok := existing[meteoFileKey{f.Meta.Variable, f.Meta.ValidTime}]; !ok {
			newFiles = append(newFiles, f)
		}
	}
	if len(newFiles) == 0 && !needGrid {
		logg.Info(ctx, "All forecast files already stored", "referenceTime", manifest.ReferenceTime)
		return nil
	}
	manifest.Files = newFiles

	logg.Info(ctx, "Downloading new forecast files",
		"referenceTime", manifest.ReferenceTime,
		"count", len(manifest.Files))

	// Stage 3: download only the new files (long operation, no DB lock held).
	result, err := DownloadForecast(ctx, manifest)
	if err != nil {
		return fmt.Errorf("download forecast: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(result.Dir); removeErr != nil {
			logg.Error(ctx, "Failed to remove forecast temp dir", "path", result.Dir, "err", removeErr)
		}
	}()

	// Stage 4: atomically write all new blobs, forecast_files, and grid constants.
	err = d.d.WithTx(ctx, func(tx *db.Queries) error {
		for _, f := range result.Files {
			absPath := filepath.Join(result.Dir, f.Path)
			content, readErr := os.ReadFile(absPath)
			if readErr != nil {
				return fmt.Errorf("read %s: %w", absPath, readErr)
			}

			b, blobErr := blob.Create(ctx, tx, content, blob.CompressionNone)
			if blobErr != nil {
				return fmt.Errorf("create blob for %s: %w", f.Path, blobErr)
			}

			if _, dbErr := tx.CreateForecastFile(ctx, db.CreateForecastFileParams{
				CreatedAt:     time.Now(),
				ReferenceTime: result.ReferenceTime,
				ValidTime:     f.Meta.ValidTime,
				Variable:      f.Meta.Variable,
				BoundsMinLat:  nullFloat(f.Meta.BoundsMinLat),
				BoundsMinLon:  nullFloat(f.Meta.BoundsMinLon),
				BoundsMaxLat:  nullFloat(f.Meta.BoundsMaxLat),
				BoundsMaxLon:  nullFloat(f.Meta.BoundsMaxLon),
				BlobID:        b.ID,
			}); dbErr != nil {
				return fmt.Errorf("create forecast_file record for %s validTime=%s: %w",
					f.Meta.Variable, f.Meta.ValidTime.Format(time.RFC3339), dbErr)
			}
		}

		if needGrid {
			gridPath := filepath.Join(result.Dir, result.GridConstantsPath)
			gridContent, readErr := os.ReadFile(gridPath)
			if readErr != nil {
				return fmt.Errorf("read grid constants: %w", readErr)
			}

			b, blobErr := blob.Create(ctx, tx, gridContent, blob.CompressionNone)
			if blobErr != nil {
				return fmt.Errorf("create blob for grid constants: %w", blobErr)
			}

			if _, dbErr := tx.CreateForecastGrid(ctx, db.CreateForecastGridParams{
				CreatedAt:     time.Now(),
				ReferenceTime: result.ReferenceTime,
				BlobID:        b.ID,
			}); dbErr != nil {
				return fmt.Errorf("create forecast_grid record: %w", dbErr)
			}
			logg.Info(ctx, "Stored grid constants", "referenceTime", result.ReferenceTime)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("write forecast to database: %w", err)
	}

	logg.Info(ctx, "Stored new forecast data", "referenceTime", result.ReferenceTime, "fileCount", len(result.Files))
	return nil
}

// nullFloat converts a float64 to sql.NullFloat64, treating NaN as invalid/null.
func nullFloat(v float64) sql.NullFloat64 {
	if math.IsNaN(v) {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: v, Valid: true}
}
