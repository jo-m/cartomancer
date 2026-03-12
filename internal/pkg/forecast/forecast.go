// Package forecast provides functionality for downloading ICON-CH1-EPS weather
// forecast data from the Swiss government STAC API.
//
// Call GetNewestForecast to retrieve metadata for the newest available run, then
// optionally filter the returned file list and pass it to DownloadForecast to
// stage the corresponding GRIB2 files locally. Download is a convenience wrapper
// that performs both steps in sequence.
//
// Docs: https://opendatadocs.meteoswiss.ch/de/e-forecast-data/e2-e3-numerical-weather-forecasting-model
// STAC Browser: https://data.geo.admin.ch/browser/index.html#/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1
package forecast

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"jo-m.ch/go/detour/internal/pkg/forecast/stac"
	"jo-m.ch/go/detour/internal/pkg/forecast/vars"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

const (
	horizConstAssetKey = "horizontal_constants_icon-ch1-eps.grib2"
	vertConstAssetKey  = "vertical_constants_icon-ch1-eps.grib2"

	// NoHorizonLimit can be passed as maxHorizon to [GetNewestForecast] to include all
	// available forecast horizons without restriction.
	NoHorizonLimit = time.Duration(math.MaxInt64)
)

// FileMeta holds the meteorological metadata shared between [ForecastFile] and
// [DownloadedFile].
type FileMeta struct {
	// Variable is the forecast variable name (e.g., "U_10M", "TOT_PREC").
	Variable string
	// Horizon is the duration from the reference time to the forecast valid time.
	Horizon time.Duration
	// ValidTime is the time at which the forecast values are valid.
	ValidTime time.Time
	// Perturbed indicates whether this is a perturbed ensemble member.
	Perturbed bool
	// BoundsMinLat is the southern latitude of the spatial domain (WGS84).
	// NaN when not available.
	BoundsMinLat float64
	// BoundsMinLon is the western longitude of the spatial domain (WGS84).
	// NaN when not available.
	BoundsMinLon float64
	// BoundsMaxLat is the northern latitude of the spatial domain (WGS84).
	// NaN when not available.
	BoundsMaxLat float64
	// BoundsMaxLon is the eastern longitude of the spatial domain (WGS84).
	// NaN when not available.
	BoundsMaxLon float64
}

// ForecastFile is a single GRIB2 file available in a forecast run, as returned
// by [GetNewestForecast] before any files have been downloaded.
type ForecastFile struct {
	// Meta contains the meteorological metadata for this file.
	Meta FileMeta
	// URL is the HTTPS download address for this file.
	URL string
}

// DownloadedFile represents a single downloaded GRIB2 file with its parsed metadata.
type DownloadedFile struct {
	// Meta contains the meteorological metadata for this file.
	Meta FileMeta
	// Path is the path to the downloaded file, relative to the containing [DownloadResult].Dir.
	Path string
}

// ForecastManifest is the result of [GetNewestForecast], listing available
// GRIB2 files for the newest forecast run without downloading them. The caller
// may filter or reorder Files before passing the manifest to [DownloadForecast].
type ForecastManifest struct {
	// ReferenceTime is the model run initialisation time.
	ReferenceTime time.Time
	// Files contains one entry per available GRIB2 file matching the query.
	Files []ForecastFile
	// GridConstantsURL is the download URL for the horizontal grid constants GRIB2 file.
	GridConstantsURL string
	// VertConstantsURL is the download URL for the vertical grid constants GRIB2 file.
	VertConstantsURL string
}

// DownloadResult holds the output of a [DownloadForecast] call.
type DownloadResult struct {
	// Dir is the temporary directory containing all downloaded files.
	// The caller must call os.RemoveAll(Dir) when the files are no longer needed.
	Dir string
	// ReferenceTime is the model run initialisation time.
	ReferenceTime time.Time
	// Files contains one entry per downloaded GRIB2 file. Paths are relative to Dir.
	Files []DownloadedFile
	// GridConstantsPath is the path to the horizontal grid constants GRIB2 file,
	// relative to Dir.
	GridConstantsPath string
	// VertConstantsPath is the path to the vertical grid constants GRIB2 file,
	// relative to Dir.
	VertConstantsPath string
}

// GetNewestForecast queries the STAC API for the newest available forecast run
// and returns a [ForecastManifest] listing matching files without downloading them.
//
// Only files whose horizon does not exceed maxHorizon and whose perturbed flag
// matches the perturbed parameter are included. The caller may further filter or
// reorder [ForecastManifest].Files before passing the manifest to [DownloadForecast].
func GetNewestForecast(ctx context.Context, variables []vars.Variable, maxHorizon time.Duration, perturbed bool) (*ForecastManifest, error) {
	paramNames := make([]string, len(variables))
	for i, v := range variables {
		paramNames[i] = v.Name
	}
	logg.Debug(ctx, "Fetching STAC items", "variables", paramNames, "maxHorizon", maxHorizon, "perturbed", perturbed)
	items, refTime, err := stac.FetchItemsForVariables(ctx, vars.CollectionBaseURL, paramNames)
	if err != nil {
		return nil, fmt.Errorf("fetching STAC items: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no forecast items found for variables %v", variables)
	}

	coll, err := stac.FetchJSON[stac.Collection](ctx, vars.CollectionBaseURL)
	if err != nil {
		return nil, fmt.Errorf("fetching STAC collection: %w", err)
	}

	horizAsset, ok := coll.Assets[horizConstAssetKey]
	if !ok {
		return nil, fmt.Errorf("asset %q not found in collection", horizConstAssetKey)
	}
	vertAsset, ok := coll.Assets[vertConstAssetKey]
	if !ok {
		return nil, fmt.Errorf("asset %q not found in collection", vertConstAssetKey)
	}

	var files []ForecastFile
	for _, item := range items {
		horizon, parseErr := stac.ParseISO8601Duration(item.Properties.Horizon)
		if parseErr != nil {
			return nil, fmt.Errorf("parsing horizon for item %s: %w", item.ID, parseErr)
		}

		if horizon > maxHorizon {
			logg.Trace(ctx, "Skipping item: horizon exceeds limit",
				"variable", item.Properties.Variable,
				"horizon", horizon,
				"maxHorizon", maxHorizon)
			continue
		}
		if item.Properties.Perturbed != perturbed {
			logg.Trace(ctx, "Skipping item: perturbed mismatch",
				"variable", item.Properties.Variable,
				"perturbed", item.Properties.Perturbed)
			continue
		}

		var assetURL string
		for _, asset := range item.Assets {
			assetURL = asset.Href
			break
		}
		if assetURL == "" {
			logg.Debug(ctx, "Skipping item: no asset URL", "id", item.ID)
			continue
		}

		f := ForecastFile{
			URL: assetURL,
			Meta: FileMeta{
				Variable:  item.Properties.Variable,
				Horizon:   horizon,
				ValidTime: refTime.Add(horizon),
				Perturbed: item.Properties.Perturbed,
			},
		}
		parseBBox(item, &f.Meta)
		files = append(files, f)
	}

	logg.Info(ctx, "Fetched forecast manifest", "referenceTime", refTime, "count", len(files))
	return &ForecastManifest{
		ReferenceTime:    refTime,
		Files:            files,
		GridConstantsURL: horizAsset.Href,
		VertConstantsURL: vertAsset.Href,
	}, nil
}

// DownloadForecast downloads the files listed in manifest to a newly created
// temporary directory. manifest.Files may be reduced or reordered by the caller
// before passing. The caller is responsible for removing Dir when finished
// (os.RemoveAll).
//
// Returns an error if any download fails; in that case the temporary directory
// is removed before returning.
func DownloadForecast(ctx context.Context, manifest *ForecastManifest) (*DownloadResult, error) {
	dir, err := os.MkdirTemp("", "forecast-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	const gridConstName = "horiz_const.grib2"
	logg.Debug(ctx, "Downloading horizontal grid constants", "dest", gridConstName)
	if err := downloadFile(ctx, manifest.GridConstantsURL, filepath.Join(dir, gridConstName)); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("downloading horizontal grid constants: %w", err)
	}

	const vertConstName = "vert_const.grib2"
	logg.Debug(ctx, "Downloading vertical grid constants", "dest", vertConstName)
	if err := downloadFile(ctx, manifest.VertConstantsURL, filepath.Join(dir, vertConstName)); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("downloading vertical grid constants: %w", err)
	}

	logg.Info(ctx, "Downloading forecast files", "referenceTime", manifest.ReferenceTime, "count", len(manifest.Files))

	var downloaded []DownloadedFile
	for i, mf := range manifest.Files {
		relPath := fmt.Sprintf("%04d.grib2", i)
		logg.Debug(ctx, "Downloading forecast file",
			"variable", mf.Meta.Variable,
			"horizon", mf.Meta.Horizon,
			"perturbed", mf.Meta.Perturbed,
			"dest", relPath)
		if downloadErr := downloadFile(ctx, mf.URL, filepath.Join(dir, relPath)); downloadErr != nil {
			_ = os.RemoveAll(dir)
			return nil, fmt.Errorf("downloading %s/%s: %w", mf.Meta.Variable, mf.Meta.Horizon, downloadErr)
		}

		downloaded = append(downloaded, DownloadedFile{
			Meta: mf.Meta,
			Path: relPath,
		})
	}

	logg.Info(ctx, "Downloaded forecast files", "referenceTime", manifest.ReferenceTime, "count", len(downloaded))
	return &DownloadResult{
		Dir:               dir,
		ReferenceTime:     manifest.ReferenceTime,
		Files:             downloaded,
		GridConstantsPath: gridConstName,
		VertConstantsPath: vertConstName,
	}, nil
}

// Download is a convenience wrapper that calls [GetNewestForecast] followed by
// [DownloadForecast]. Files are written to a newly created temporary directory.
// The caller is responsible for removing Dir when finished (os.RemoveAll).
//
// Only files whose horizon does not exceed maxHorizon and whose perturbed flag
// matches the perturbed parameter are downloaded.
//
// Returns an error if any step fails; in that case the temporary directory
// is removed before returning.
func Download(ctx context.Context, variables []vars.Variable, maxHorizon time.Duration, perturbed bool) (*DownloadResult, error) {
	manifest, err := GetNewestForecast(ctx, variables, maxHorizon, perturbed)
	if err != nil {
		return nil, err
	}
	return DownloadForecast(ctx, manifest)
}

// FetchLatestReferenceTime queries the STAC catalog and returns the model
// initialisation time of the newest available forecast run. It returns the
// zero time with a nil error when the catalog contains no items.
func FetchLatestReferenceTime(ctx context.Context) (time.Time, error) {
	// Pass nil variables: FetchItemsForVariables scans all items to find the
	// max reference time before filtering, so the reference time is accurate
	// regardless of the variable filter.
	_, refTime, err := stac.FetchItemsForVariables(ctx, vars.CollectionBaseURL, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("fetching STAC items: %w", err)
	}
	return refTime, nil
}

// parseBBox fills the bounding-box fields of m from the STAC item's BBox slice,
// which follows the GeoJSON convention [min_lon, min_lat, max_lon, max_lat].
// Fields are set to NaN when the bbox is absent or malformed.
func parseBBox(item stac.Item, m *FileMeta) {
	m.BoundsMinLat = math.NaN()
	m.BoundsMinLon = math.NaN()
	m.BoundsMaxLat = math.NaN()
	m.BoundsMaxLon = math.NaN()
	if len(item.BBox) == 4 {
		m.BoundsMinLon = item.BBox[0]
		m.BoundsMinLat = item.BBox[1]
		m.BoundsMaxLon = item.BBox[2]
		m.BoundsMaxLat = item.BBox[3]
	}
}

// downloadFile downloads the resource at url and writes it to destPath.
func downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}
