package meteo

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/meteo/vars"
)

// FileValidityDuration is the duration for which a single forecast file is
// considered valid, forming the half-open interval [valid_time, valid_time + FileValidityDuration).
const FileValidityDuration = 1 * time.Hour

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

// Run implements [jobs.Job].
// It checks whether the newest forecast run available online is already stored
// in the database. If the reference time already exists, it returns early.
// Otherwise it downloads all files for that run and commits them atomically.
func (d *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Stage 1: fetch the newest reference time from the STAC API (lightweight).
	latestRefTime, err := FetchLatestReferenceTime(ctx)
	if err != nil {
		return fmt.Errorf("fetch latest reference time: %w", err)
	}
	if latestRefTime.IsZero() {
		logg.Info(ctx, "No forecast runs available online")
		return nil
	}

	// Stage 2: check if a forecast row already exists for this reference time.
	exists, err := d.d.QueryRO().ForecastExistsForReferenceTime(ctx, latestRefTime)
	if err != nil {
		return fmt.Errorf("check existing forecast: %w", err)
	}
	if exists != 0 {
		logg.Info(ctx, "Forecast already stored", "referenceTime", latestRefTime)
		return nil
	}

	// Stage 3: fetch full manifest and download all files.
	manifest, err := GetNewestForecast(ctx, DownloadVariables, NoHorizonLimit, false)
	if err != nil {
		return fmt.Errorf("fetch forecast manifest: %w", err)
	}

	// Re-check: the actual reference time may differ from the collection extent
	// estimate (e.g. the newest model run is still uploading), so verify the
	// resolved reference time is not already stored.
	if !manifest.ReferenceTime.Equal(latestRefTime) {
		exists, dbErr := d.d.QueryRO().ForecastExistsForReferenceTime(ctx, manifest.ReferenceTime)
		if dbErr != nil {
			return fmt.Errorf("re-check existing forecast: %w", dbErr)
		}
		if exists != 0 {
			logg.Info(ctx, "Forecast already stored (after probing)", "referenceTime", manifest.ReferenceTime)
			return nil
		}
	}

	logg.Info(ctx, "Downloading new forecast",
		"referenceTime", manifest.ReferenceTime,
		"fileCount", len(manifest.Files))

	result, err := DownloadForecast(ctx, manifest)
	if err != nil {
		return fmt.Errorf("download forecast: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(result.Dir); removeErr != nil {
			logg.Error(ctx, "Failed to remove forecast temp dir", "path", result.Dir, "err", removeErr)
		}
	}()

	// Consistency check: every downloaded file must report the same reference
	// time as the manifest.
	for _, f := range result.Files {
		if !f.Meta.ReferenceTime.Equal(manifest.ReferenceTime) {
			return fmt.Errorf(
				"reference time mismatch for %s: manifest=%s, file=%s",
				f.Meta.Variable,
				manifest.ReferenceTime.Format(time.RFC3339),
				f.Meta.ReferenceTime.Format(time.RFC3339),
			)
		}
	}

	// Stage 4: atomically write forecast row and all forecast_files.
	err = d.d.WithTx(ctx, func(tx *db.Queries) error {
		gridPath := filepath.Join(result.Dir, result.GridConstantsPath)
		gridContent, readErr := os.ReadFile(gridPath)
		if readErr != nil {
			return fmt.Errorf("read horizontal grid constants: %w", readErr)
		}

		vertGridPath := filepath.Join(result.Dir, result.VertConstantsPath)
		vertGridContent, readErr := os.ReadFile(vertGridPath)
		if readErr != nil {
			return fmt.Errorf("read vertical grid constants: %w", readErr)
		}

		// Use the bounding box from the first file (all files in a run
		// share the same spatial domain).
		var boundsMinLat, boundsMinLon, boundsMaxLat, boundsMaxLon float64
		boundsMinLat = math.NaN()
		boundsMinLon = math.NaN()
		boundsMaxLat = math.NaN()
		boundsMaxLon = math.NaN()
		if len(result.Files) > 0 {
			boundsMinLat = result.Files[0].Meta.BoundsMinLat
			boundsMinLon = result.Files[0].Meta.BoundsMinLon
			boundsMaxLat = result.Files[0].Meta.BoundsMaxLat
			boundsMaxLon = result.Files[0].Meta.BoundsMaxLon
		}

		forecastRow, dbErr := tx.CreateForecast(ctx, db.CreateForecastParams{
			CreatedAt:          time.Now(),
			ReferenceTime:      result.ReferenceTime,
			BoundsMinLat:       nullFloat(boundsMinLat),
			BoundsMinLon:       nullFloat(boundsMinLon),
			BoundsMaxLat:       nullFloat(boundsMaxLat),
			BoundsMaxLon:       nullFloat(boundsMaxLon),
			HorizontalGridFile: gridContent,
			VerticalGridFile:   vertGridContent,
			Attribution:        "MeteoSwiss (CC-BY)",
			AttributionHref:    "https://www.meteoswiss.admin.ch/",
		})
		if dbErr != nil {
			return fmt.Errorf("create forecast record: %w", dbErr)
		}
		logg.Info(ctx, "Stored forecast", "referenceTime", result.ReferenceTime)

		for _, f := range result.Files {
			absPath := filepath.Join(result.Dir, f.Path)
			content, readErr := os.ReadFile(absPath)
			if readErr != nil {
				return fmt.Errorf("read %s: %w", absPath, readErr)
			}

			if _, dbErr := tx.CreateForecastFile(ctx, db.CreateForecastFileParams{
				ValidTime:      f.Meta.ValidTime,
				ValidUntilTime: f.Meta.ValidTime.Add(FileValidityDuration),
				Variable:       f.Meta.Variable,
				File:           content,
				ForecastID:     forecastRow.ID,
			}); dbErr != nil {
				return fmt.Errorf("create forecast_file record for %s validTime=%s: %w",
					f.Meta.Variable, f.Meta.ValidTime.Format(time.RFC3339), dbErr)
			}
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
