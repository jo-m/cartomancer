// Package forecast provides functionality for downloading ICON-CH1-EPS weather
// forecast data from the Swiss government STAC API.
//
// Usage is a two-step process: first call FetchVariables to obtain the list of
// available meteorological variables and their metadata; then call Download with
// a selection of variable names to stage the corresponding GRIB2 files locally.
//
// Docs: https://opendatadocs.meteoswiss.ch/de/e-forecast-data/e2-e3-numerical-weather-forecasting-model
// STAC Browser: https://data.geo.admin.ch/browser/index.html#/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1
package forecast

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"jo-m.ch/go/detour/internal/pkg/logg"
)

const (
	collectionBaseURL  = "https://data.geo.admin.ch/api/stac/v0.9/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1"
	csvAssetKey        = "params_icon-ch1-eps.csv"
	horizConstAssetKey = "horizontal_constants_icon-ch1-eps.grib2"
	itemsPageSize      = 1000
)

// Variable describes a meteorological forecast variable available in the
// ICON-CH1-EPS model.
type Variable struct {
	// Parameter is the short parameter code (e.g., "T_2M").
	Parameter string
	// LongName is the human-readable description (e.g., "2m Temperature").
	LongName string
	// Unit is the standard physical unit (e.g., "K").
	Unit string
	// Level indicates whether the variable is "Single Level" or "Multi Level".
	Level string
	// VerticalCoordinate describes the vertical coordinate system.
	VerticalCoordinate string
	// Horizon is the range of available forecast horizons (e.g., "0-33h").
	Horizon string
	// TemporalAggregation is the aggregation method (e.g., "Instant", "Average").
	TemporalAggregation string
	// AggregationStart is the start reference for temporal aggregation.
	AggregationStart string
}

// FetchVariables downloads and parses the parameter CSV from the STAC collection,
// returning the list of available forecast variables with their metadata.
//
// The CSV is fetched at runtime from the collection's asset href, so the caller
// must have network access to the Swiss government STAC API.
func FetchVariables(ctx context.Context) ([]Variable, error) {
	coll, err := fetchJSON[stacCollection](ctx, collectionBaseURL)
	if err != nil {
		return nil, fmt.Errorf("fetching STAC collection: %w", err)
	}

	csvAsset, ok := coll.Assets[csvAssetKey]
	if !ok {
		return nil, fmt.Errorf("asset %q not found in collection", csvAssetKey)
	}

	return downloadAndParseCSV(ctx, csvAsset.Href)
}

// DownloadedFile represents a single downloaded GRIB2 file with its parsed metadata.
type DownloadedFile struct {
	// Path is the absolute path to the downloaded file within the temporary directory.
	Path string
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

// DownloadResult holds the output of a [Download] call.
type DownloadResult struct {
	// Dir is the temporary directory containing all downloaded files.
	// The caller must call os.RemoveAll(Dir) when the files are no longer needed.
	Dir string
	// ReferenceTime is the model run initialisation time.
	ReferenceTime time.Time
	// Files contains one entry per downloaded GRIB2 file.
	Files []DownloadedFile
	// GridConstantsPath is the path to the horizontal constants GRIB2 file within Dir.
	GridConstantsPath string
	// VariablesCSVPath is the path to the forecast variables parameter CSV within Dir.
	VariablesCSVPath string
}

// Download fetches GRIB2 forecast files for the given variables from the newest
// available forecast run. Files are written to a newly created temporary directory.
// The caller is responsible for removing Dir when finished (os.RemoveAll).
//
// variables is a list of parameter codes as they appear in the parameter CSV
// (e.g., "T_2M", "TOT_PREC"). Both ctrl and perturb files are downloaded for
// each variable when available.
//
// Returns an error if any download fails; in that case the temporary directory
// is removed before returning.
func Download(ctx context.Context, variables []string) (*DownloadResult, error) {
	items, refTime, err := fetchItemsForVariables(ctx, variables)
	if err != nil {
		return nil, fmt.Errorf("fetching STAC items: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no forecast items found for variables %v", variables)
	}

	logg.Info(ctx, "Downloading forecast files", "referenceTime", refTime, "count", len(items))

	dir, err := os.MkdirTemp("", "forecast-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	coll, err := fetchJSON[stacCollection](ctx, collectionBaseURL)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("fetching STAC collection: %w", err)
	}

	horizAsset, ok := coll.Assets[horizConstAssetKey]
	if !ok {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("asset %q not found in collection", horizConstAssetKey)
	}
	gridConstantsPath := filepath.Join(dir, "horiz_const.grib2")
	if err := downloadFile(ctx, horizAsset.Href, gridConstantsPath); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("downloading grid constants: %w", err)
	}

	csvAsset, ok := coll.Assets[csvAssetKey]
	if !ok {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("asset %q not found in collection", csvAssetKey)
	}
	variablesCSVPath := filepath.Join(dir, "params.csv")
	if err := downloadFile(ctx, csvAsset.Href, variablesCSVPath); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("downloading variables CSV: %w", err)
	}

	var files []DownloadedFile
	for i, item := range items {
		horizon, parseErr := parseISO8601Duration(item.Properties.Horizon)
		if parseErr != nil {
			_ = os.RemoveAll(dir)
			return nil, fmt.Errorf("parsing horizon for item %s: %w", item.ID, parseErr)
		}

		var assetURL string
		for _, asset := range item.Assets {
			assetURL = asset.Href
			break
		}
		if assetURL == "" {
			continue
		}

		localPath := filepath.Join(dir, fmt.Sprintf("%04d.grib2", i))
		logg.Debug(ctx, "Downloading forecast file",
			"variable", item.Properties.Variable,
			"horizon", item.Properties.Horizon,
			"perturbed", item.Properties.Perturbed)
		if downloadErr := downloadFile(ctx, assetURL, localPath); downloadErr != nil {
			_ = os.RemoveAll(dir)
			return nil, fmt.Errorf("downloading %s/%s: %w",
				item.Properties.Variable, item.Properties.Horizon, downloadErr)
		}

		f := DownloadedFile{
			Path:      localPath,
			Variable:  item.Properties.Variable,
			Horizon:   horizon,
			ValidTime: refTime.Add(horizon),
			Perturbed: item.Properties.Perturbed,
		}
		parseBBox(item, &f)
		files = append(files, f)
	}

	logg.Info(ctx, "Downloaded forecast files", "referenceTime", refTime, "count", len(files))
	return &DownloadResult{
		Dir:               dir,
		ReferenceTime:     refTime,
		Files:             files,
		GridConstantsPath: gridConstantsPath,
		VariablesCSVPath:  variablesCSVPath,
	}, nil
}

// parseBBox fills the bounding-box fields of f from the STAC item's BBox slice,
// which follows the GeoJSON convention [min_lon, min_lat, max_lon, max_lat].
// Fields are set to NaN when the bbox is absent or malformed.
func parseBBox(item stacItem, f *DownloadedFile) {
	f.BoundsMinLat = math.NaN()
	f.BoundsMinLon = math.NaN()
	f.BoundsMaxLat = math.NaN()
	f.BoundsMaxLon = math.NaN()
	if len(item.BBox) == 4 {
		f.BoundsMinLon = item.BBox[0]
		f.BoundsMinLat = item.BBox[1]
		f.BoundsMaxLon = item.BBox[2]
		f.BoundsMaxLat = item.BBox[3]
	}
}

// downloadAndParseCSV fetches the CSV at href and parses it into Variable records.
func downloadAndParseCSV(ctx context.Context, href string) ([]Variable, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, href, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching CSV", resp.StatusCode)
	}

	r := csv.NewReader(resp.Body)
	// Skip the header row.
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	var vars []Variable
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV record: %w", err)
		}
		if len(rec) < 8 {
			continue
		}
		vars = append(vars, Variable{
			Parameter:           rec[0],
			LongName:            rec[1],
			Unit:                rec[2],
			Level:               rec[3],
			VerticalCoordinate:  rec[4],
			Horizon:             rec[5],
			TemporalAggregation: rec[6],
			AggregationStart:    rec[7],
		})
	}
	return vars, nil
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
