package meteo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo/vars"
)

const (
	// downloaderTimeout is the maximum time the downloader job is allowed to run.
	downloaderTimeout = 10 * time.Minute
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

// Run implements [jobs.Job].
// It checks whether the newest forecast run available online is already stored
// in the database. If the reference time already exists but is incomplete
// (missing variables), it fetches the manifest and augments with missing files.
// Otherwise it downloads all files for that run and commits them atomically.
func (d *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, downloaderTimeout)
	defer cancel()

	// Stage 1: fetch the newest reference time from the STAC API (lightweight).
	latestRefTime, err := FetchLatestReferenceTime(ctx)
	if err != nil {
		return fmt.Errorf("fetch latest reference time: %w", err)
	}
	if latestRefTime.IsZero() {
		logg.Info(ctx, "no forecast runs available online")
		return nil
	}

	// Stage 2: check if a forecast row already exists for this reference time
	// and whether it has all expected variables.
	exists, err := d.d.QueryRO().ForecastExistsForReferenceTime(ctx, latestRefTime)
	if err != nil {
		return fmt.Errorf("check existing forecast: %w", err)
	}
	if exists != 0 {
		complete, checkErr := d.isForecastComplete(ctx, latestRefTime)
		if checkErr != nil {
			return fmt.Errorf("check forecast completeness: %w", checkErr)
		}
		if complete {
			logg.Info(ctx, "forecast already stored and complete", "referenceTime", latestRefTime)
			return nil
		}
		logg.Info(ctx, "forecast incomplete, will attempt to augment", "referenceTime", latestRefTime)
	}

	// Stage 3: fetch full manifest and download all/missing files.
	manifest, err := GetNewestForecast(ctx, DownloadVariables, noHorizonLimit, false)
	if err != nil {
		return fmt.Errorf("fetch forecast manifest: %w", err)
	}

	// Check if the manifest's resolved reference time already exists in the DB.
	forecastRow, err := d.d.QueryRO().GetForecastByReferenceTime(ctx, manifest.ReferenceTime)
	if errors.Is(err, sql.ErrNoRows) {
		return d.storeNewForecast(ctx, manifest)
	}
	if err != nil {
		return fmt.Errorf("check existing forecast for resolved ref time: %w", err)
	}

	// Forecast row exists -- augment with any missing files.
	return d.augmentForecast(ctx, manifest, forecastRow.ID)
}

// isForecastComplete returns true if the forecast for the given reference time
// has files for all expected variables in [DownloadVariables].
func (d *Downloader) isForecastComplete(ctx context.Context, refTime time.Time) (bool, error) {
	count, err := d.d.QueryRO().CountDistinctForecastVariables(ctx, refTime)
	if err != nil {
		return false, err
	}
	return count >= int64(len(DownloadVariables)), nil
}

// storeNewForecast downloads all files from the manifest and creates a new
// forecast row with all files atomically.
func (d *Downloader) storeNewForecast(ctx context.Context, manifest *ForecastManifest) error {
	logg.Info(ctx, "downloading new forecast",
		"referenceTime", manifest.ReferenceTime,
		"fileCount", len(manifest.Files))

	result, err := DownloadForecast(ctx, manifest)
	if err != nil {
		return fmt.Errorf("download forecast: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(result.Dir); removeErr != nil {
			logg.Error(ctx, "failed to remove forecast temp dir", "path", result.Dir, "err", removeErr)
		}
	}()

	if err := verifyReferenceTime(result.Files, manifest.ReferenceTime); err != nil {
		return err
	}

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
			Attribution:        attribution,
			AttributionHref:    attributionHref,
		})
		if dbErr != nil {
			return fmt.Errorf("create forecast record: %w", dbErr)
		}

		return insertFiles(ctx, tx, result.Dir, result.Files, forecastRow.ID)
	})
	if err != nil {
		return fmt.Errorf("write forecast to database: %w", err)
	}

	logg.Info(ctx, "stored new forecast data", "referenceTime", result.ReferenceTime, "fileCount", len(result.Files))
	return nil
}

// augmentForecast downloads files that are present in the manifest but missing
// from the existing forecast row and inserts them.
func (d *Downloader) augmentForecast(ctx context.Context, manifest *ForecastManifest, forecastID int64) error {
	existingKeys, err := d.d.QueryRO().ListForecastFileKeys(ctx, forecastID)
	if err != nil {
		return fmt.Errorf("list existing forecast file keys: %w", err)
	}

	existing := make(map[string]struct{}, len(existingKeys))
	for _, k := range existingKeys {
		existing[fileKey(k.Variable, k.ValidTime)] = struct{}{}
	}

	var missing []ForecastFile
	for _, f := range manifest.Files {
		if _, ok := existing[fileKey(f.Meta.Variable, f.Meta.ValidTime)]; !ok {
			missing = append(missing, f)
		}
	}

	if len(missing) == 0 {
		logg.Info(ctx, "forecast is complete, nothing to augment", "referenceTime", manifest.ReferenceTime)
		return nil
	}

	logg.Info(ctx, "augmenting incomplete forecast",
		"referenceTime", manifest.ReferenceTime,
		"existingFiles", len(existingKeys),
		"missingFiles", len(missing))

	dir, err := os.MkdirTemp("", "forecast-augment-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(dir); removeErr != nil {
			logg.Error(ctx, "failed to remove forecast augment temp dir", "path", dir, "err", removeErr)
		}
	}()

	var downloaded []DownloadedFile
	for i, mf := range missing {
		relPath := fmt.Sprintf("%04d.grib2", i)
		if downloadErr := downloadFile(ctx, mf.URL, filepath.Join(dir, relPath)); downloadErr != nil {
			return fmt.Errorf("downloading %s/%s: %w", mf.Meta.Variable, mf.Meta.Horizon, downloadErr)
		}
		downloaded = append(downloaded, DownloadedFile{Meta: mf.Meta, Path: relPath})
	}

	if err := verifyReferenceTime(downloaded, manifest.ReferenceTime); err != nil {
		return err
	}

	err = d.d.WithTx(ctx, func(tx *db.Queries) error {
		return insertFiles(ctx, tx, dir, downloaded, forecastID)
	})
	if err != nil {
		return fmt.Errorf("write augmented files to database: %w", err)
	}

	logg.Info(ctx, "augmented forecast", "referenceTime", manifest.ReferenceTime, "addedFiles", len(downloaded))
	return nil
}

// fileKey returns a map key for a (variable, validTime) pair.
func fileKey(variable string, validTime time.Time) string {
	return variable + "|" + validTime.Format(time.RFC3339)
}

// verifyReferenceTime checks that all downloaded files report the expected reference time.
func verifyReferenceTime(files []DownloadedFile, expected time.Time) error {
	for _, f := range files {
		if !f.Meta.ReferenceTime.Equal(expected) {
			return fmt.Errorf(
				"reference time mismatch for %s: expected=%s, file=%s",
				f.Meta.Variable,
				expected.Format(time.RFC3339),
				f.Meta.ReferenceTime.Format(time.RFC3339),
			)
		}
	}
	return nil
}

// insertFiles writes downloaded GRIB2 files as forecast_file rows within an
// existing transaction.
func insertFiles(ctx context.Context, tx *db.Queries, dir string, files []DownloadedFile, forecastID int64) error {
	for _, f := range files {
		absPath := filepath.Join(dir, f.Path)
		content, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", absPath, readErr)
		}

		if _, dbErr := tx.CreateForecastFile(ctx, db.CreateForecastFileParams{
			ValidTime:      f.Meta.ValidTime,
			ValidUntilTime: f.Meta.ValidTime.Add(fileValidityDuration),
			Variable:       f.Meta.Variable,
			File:           content,
			ForecastID:     forecastID,
		}); dbErr != nil {
			return fmt.Errorf("create forecast_file record for %s validTime=%s: %w",
				f.Meta.Variable, f.Meta.ValidTime.Format(time.RFC3339), dbErr)
		}
	}
	return nil
}

// nullFloat converts a float64 to sql.NullFloat64, treating NaN as invalid/null.
func nullFloat(v float64) sql.NullFloat64 {
	if math.IsNaN(v) {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: v, Valid: true}
}
