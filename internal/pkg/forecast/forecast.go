// Package forecast provides functionality for downloading ICON-CH1-EPS weather
// forecast data from the Swiss government STAC API.
//
// Usage is a two-step process: first call FetchVariables to obtain the list of
// available meteorological variables and their metadata; then call Download with
// a selection of variable names to stage the corresponding GRIB2 files locally.
package forecast

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	collectionBaseURL = "https://data.geo.admin.ch/api/stac/v0.9/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1"
	csvAssetKey       = "params_icon-ch1-eps.csv"
	itemsPageSize     = 100
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
	// Horizon is the forecast time horizon.
	Horizon string
	// TemporalAggregation is the aggregation method (e.g., "Instant", "Average").
	TemporalAggregation string
	// AggregationStart is the start reference for temporal aggregation (e.g., "Reference Time" or "-").
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

// FileSet maps variable-type keys to local GRIB2 file paths.
// Keys follow the pattern "{VARIABLE}-{type}", for example "T_2M-ctrl" or "TOT_PREC-perturb".
type FileSet map[string]string

// DownloadResult holds the output of a Download call.
type DownloadResult struct {
	// Dir is the temporary directory containing all downloaded files.
	// The caller must call os.RemoveAll(Dir) when the files are no longer needed.
	Dir string
	// Files maps variable-type keys to absolute file paths within Dir.
	// Keys are of the form "{VARIABLE}-{type}" (e.g., "T_2M-ctrl").
	Files FileSet
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
	items, forecastTime, err := fetchItemsForVariables(ctx, variables)
	if err != nil {
		return nil, fmt.Errorf("fetching STAC items (forecast %s): %w", forecastTime, err)
	}

	dir, err := os.MkdirTemp("", "forecast-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	files := make(FileSet, len(items))
	for _, item := range items {
		for assetKey, asset := range item.Assets {
			varKey := assetKeyToVarKey(assetKey)
			if varKey == "" {
				continue
			}
			localPath := filepath.Join(dir, filepath.Base(assetKey))
			if err := downloadFile(ctx, asset.Href, localPath); err != nil {
				_ = os.RemoveAll(dir)
				return nil, fmt.Errorf("downloading %s: %w", assetKey, err)
			}
			files[varKey] = localPath
		}
	}

	return &DownloadResult{Dir: dir, Files: files}, nil
}

// assetKeyToVarKey converts an asset filename like
// "icon-ch1-eps-202603101800-0-t_2m-ctrl.grib2" to a variable-type key like
// "T_2M-ctrl".
//
// Returns an empty string if the key does not match the expected format.
func assetKeyToVarKey(assetKey string) string {
	name := strings.TrimSuffix(assetKey, ".grib2")
	// Format: icon-ch1-eps-{datetime}-0-{variable}-{type}
	// Split on "-0-" to isolate the variable-type suffix.
	_, varType, found := strings.Cut(name, "-0-")
	if !found {
		return ""
	}
	// Split on the last "-" to separate variable name from ensemble type.
	lastDash := strings.LastIndex(varType, "-")
	if lastDash < 0 {
		return strings.ToUpper(varType)
	}
	variable := strings.ToUpper(varType[:lastDash])
	kind := varType[lastDash+1:]
	return variable + "-" + kind
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
