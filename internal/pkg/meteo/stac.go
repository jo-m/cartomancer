package meteo

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"jo-m.ch/go/detour/internal/pkg/attribute"
	"jo-m.ch/go/detour/internal/pkg/geoadmin"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/meteo/collection"
)

// https://data.geo.admin.ch/browser/index.html#/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1?.language=en
// https://opendatadocs.meteoswiss.ch/e-forecast-data/e2-e3-numerical-weather-forecasting-model
// https://data.geo.admin.ch/api/stac/v1/search?collections=ch.meteoschweiz.ogd-forecasting-icon-ch1
// https://data.geo.admin.ch/api/stac/v1/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1
const (
	// attribution is the data attribution label for MeteoSwiss forecast data.
	attribution = "MeteoSwiss (CC-BY)"
	// attributionHref is the URL for MeteoSwiss attribution.
	attributionHref = "https://www.meteoswiss.admin.ch/"

	// collectionHorizon is used to compute the latest reference time from the
	// collection temporal extent.
	collectionHorizon = 33 * time.Hour

	// fileValidityDuration is the duration for which a single forecast file is
	// considered valid, forming the half-open interval [valid_time, valid_time + fileValidityDuration).
	fileValidityDuration = 1 * time.Hour
	// modelRunInterval is the time between consecutive ICON-CH1-EPS model runs.
	modelRunInterval = 6 * time.Hour
)

// DataAttribution is the TASL attribution for MeteoSwiss ICON-CH1-EPS forecast data.
// Verified by TestOnlineStacLicense.
var DataAttribution = attribute.Attribution{
	What:       "Weather Forecast Data (Switzerland)",
	Title:      "ICON-CH1-EPS Forecast Data",
	Author:     "MeteoSwiss",
	Source:     attributionHref,
	License:    "CC-BY",
	LicenseURL: "https://creativecommons.org/licenses/by/4.0/",
}

// newClient creates a [geoadmin.Client] pointing at the MeteoSwiss STAC API.
func newClient() *geoadmin.Client {
	ret := geoadmin.NewClient(geoadmin.BaseURL)
	return &ret
}

// sortFeatures sorts forecast features by reference datetime, valid datetime,
// variable name, and perturbed flag.
func sortFeatures(features []geoadmin.Feature) {
	slices.SortFunc(features, func(a, b geoadmin.Feature) int {
		ap, bp := a.Properties.Forecast(), b.Properties.Forecast()

		if c := ap.ReferenceDatetime.Compare(bp.ReferenceDatetime); c != 0 {
			return c
		}
		if c := ap.Datetime.Compare(bp.Datetime); c != 0 {
			return c
		}
		if c := cmp.Compare(ap.Variable, bp.Variable); c != 0 {
			return c
		}
		if !ap.Perturbed && bp.Perturbed {
			return -1
		}
		if ap.Perturbed && !bp.Perturbed {
			return 1
		}
		return 0
	})
}

// newestReferenceTime returns the newest forecast reference datetime from the
// collection's temporal extent. Returns the zero time if the extent is empty.
func newestReferenceTime(c *geoadmin.Collection) time.Time {
	if len(c.Extent.Temporal.Interval) == 0 {
		return time.Time{}
	}

	return c.Extent.Temporal.Interval[0][1].Add(-collectionHorizon)
}

// maxRefTimeRetries is the number of older reference times to probe when
// the newest computed reference time has no items yet (e.g. because the
// model run is still in progress).
const maxRefTimeRetries = 8

// fetchItemsForVariables fetches the STAC collection to determine the newest
// forecast reference datetime from the temporal extent, then uses the Search API
// to retrieve items matching each requested variable for that reference time.
//
// Because the collection's temporal extent may advertise a model run that is
// still being uploaded, the function probes a single variable first. If no items
// are returned, it steps back by [modelRunInterval] and retries up to
// [maxRefTimeRetries] times before giving up.
//
// The returned [geoadmin.Collection] can be used by callers to extract additional
// metadata (e.g. grid constants asset URLs) without a second fetch.
//
// Returns an error if the collection has no temporal extent or if any search
// request fails.
func fetchItemsForVariables(ctx context.Context, variables []string, perturbed bool) ([]geoadmin.Feature, *geoadmin.Collection, error) {
	client := newClient()

	coll, err := client.GetCollection(ctx, collection.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching STAC collection: %w", err)
	}

	refTime := newestReferenceTime(coll)
	if refTime.IsZero() {
		return nil, coll, nil
	}

	// Probe with the first variable to find a reference time that has data.
	probeVar := strings.ToUpper(variables[0])
	for attempt := range maxRefTimeRetries {
		candidate := refTime.Add(-time.Duration(attempt) * modelRunInterval)
		candidateStr := candidate.Format(time.RFC3339)

		features, searchErr := client.SearchPost(ctx, geoadmin.SearchPostBody{
			Collections:               []string{collection.ID},
			ForecastReferenceDatetime: candidateStr,
			ForecastVariable:          probeVar,
			ForecastPerturbed:         &perturbed,
		})
		if searchErr != nil {
			return nil, nil, fmt.Errorf("probing reference time %s for variable %s: %w", candidateStr, probeVar, searchErr)
		}

		if len(features) > 0 {
			refTime = candidate
			logg.Debug(ctx, "found available reference time", "refTime", candidateStr, "attempt", attempt)
			break
		}

		logg.Debug(ctx, "no items for reference time, stepping back", "refTime", candidateStr)
		if attempt == maxRefTimeRetries-1 {
			return nil, coll, nil
		}
	}

	refTimeStr := refTime.Format(time.RFC3339)

	// Fetch all variables for the confirmed reference time.
	var all []geoadmin.Feature
	for _, v := range variables {
		features, searchErr := client.SearchPost(ctx, geoadmin.SearchPostBody{
			Collections:               []string{collection.ID},
			ForecastReferenceDatetime: refTimeStr,
			ForecastVariable:          strings.ToUpper(v),
			ForecastPerturbed:         &perturbed,
		})
		if searchErr != nil {
			return nil, nil, fmt.Errorf("searching items for variable %s: %w", v, searchErr)
		}
		all = append(all, features...)
		logg.Debug(ctx, "search returned items", "variable", v, "count", len(features))
	}

	sortFeatures(all)

	return all, coll, nil
}
