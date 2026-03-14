package vars

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"

	"jo-m.ch/go/detour/internal/pkg/meteo/stac"
)

const (
	CollectionBaseURL = "https://data.geo.admin.ch/api/stac/v0.9/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1"
	CsvAssetKey       = "params_icon-ch1-eps.csv"
)

// Variable describes a meteorological forecast variable available in the
// ICON-CH1-EPS model.
type Variable struct {
	// Name is the short parameter code (e.g., "T_2M").
	Name string
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
	coll, err := stac.FetchJSON[stac.Collection](ctx, CollectionBaseURL)
	if err != nil {
		return nil, fmt.Errorf("fetching STAC collection: %w", err)
	}

	csvAsset, ok := coll.Assets[CsvAssetKey]
	if !ok {
		return nil, fmt.Errorf("asset %q not found in collection", CsvAssetKey)
	}

	return downloadAndParseCSV(ctx, csvAsset.Href)
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
			Name:                rec[0],
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
