// Package stac provides helpers for fetching and parsing STAC (SpatioTemporal
// Asset Catalog) items from the MeteoSwiss ICON-CH1-EPS forecast API.
package stac

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"jo-m.ch/go/detour/internal/pkg/logg"
)

// https://data.geo.admin.ch/browser/index.html#/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1?.language=en
// https://data.geo.admin.ch/api/stac/static/spec/v1/api.html
// https://data.geo.admin.ch/api/stac/static/spec/v1/openapi.yaml
// https://opendatadocs.meteoswiss.ch/e-forecast-data/e2-e3-numerical-weather-forecasting-model
// https://data.geo.admin.ch/api/stac/v1/search?collections=ch.meteoschweiz.ogd-forecasting-icon-ch1
// https://data.geo.admin.ch/api/stac/v1/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1
const (
	APIBaseURL        = "https://data.geo.admin.ch/api/stac/v1"
	CollectionID      = "ch.meteoschweiz.ogd-forecasting-icon-ch1"
	CollectionHorizon = time.Hour * 33 // We use that to compute the latest reference time from collection temporal extent.
)

// GetCollectionURL returns the URL for the ICON-CH1-EPS STAC collection.
func GetCollectionURL() string {
	return fmt.Sprintf("%s/collections/%s", APIBaseURL, CollectionID)
}

func getSearchURL() string {
	return fmt.Sprintf("%s/search", APIBaseURL)
}

// Extent describes the spatial and temporal coverage of a STAC collection.
type Extent struct {
	Spatial  Spatial  `json:"spatial"`
	Temporal Temporal `json:"temporal"`
}

// Spatial describes the geographic bounding box of a STAC collection.
type Spatial struct {
	Bbox [][]float64 `json:"bbox"`
}

// Temporal describes the time interval(s) of a STAC collection.
type Temporal struct {
	Interval [][]time.Time `json:"interval"`
}

// Collection is a partial STAC Collection object.
type Collection struct {
	StacVersion string           `json:"stac_version"`
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Extent      Extent           `json:"extent"`
	License     string           `json:"license"`
	Links       []Link           `json:"links"`
	CRS         []string         `json:"crs"`
	ItemType    string           `json:"itemType"`
	Assets      map[string]Asset `json:"assets"`
}

// NewestReferenceTime returns the newest forecast reference datetime from the
// collection's temporal extent. Returns the zero time if the extent is empty.
func (c *Collection) NewestReferenceTime() time.Time {
	if len(c.Extent.Temporal.Interval) == 0 {
		return time.Time{}
	}

	if len(c.Extent.Temporal.Interval[0]) < 2 {
		return time.Time{}
	}

	return c.Extent.Temporal.Interval[0][1].Add(-CollectionHorizon)
}

// Asset is a STAC Asset with a download URL and content type.
type Asset struct {
	Href  string `json:"href"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// ItemCollection is a GeoJSON FeatureCollection of STAC Items with
// pagination links.
type ItemCollection struct {
	Features []Item `json:"features"`
	Links    []Link `json:"links"`
}

// Item is a partial STAC Item (GeoJSON Feature) carrying forecast assets.
// BBox follows the GeoJSON convention: [min_lon, min_lat, max_lon, max_lat].
type Item struct {
	ID     string           `json:"id"`
	BBox   []float64        `json:"bbox"`
	Props  ItemProps        `json:"properties"`
	Assets map[string]Asset `json:"assets"`
}

// ItemProps holds the metadata of a STAC item.
type ItemProps struct {
	// Model run initialisation time, RFC 3339.
	ReferenceDatetime time.Time `json:"forecast:reference_datetime"`
	// Timestamp of when this file is valid, RFC 3339.
	ValidDatetime time.Time `json:"datetime"`
	// Variable name.
	Variable string `json:"forecast:variable"`
	// Difference between [ReferenceDatetime] and [ValidDatetime].
	// ISO 8601 compliant duration.
	Horizon string `json:"forecast:horizon"`
	// Perturbed or control.
	Perturbed bool `json:"forecast:perturbed"`
}

// Link is a STAC hyperlink with optional fields for POST-based pagination.
type Link struct {
	Rel    string          `json:"rel"`
	Href   string          `json:"href"`
	Title  string          `json:"title,omitempty"`
	Type   string          `json:"type,omitempty"`
	Method string          `json:"method,omitempty"`
	Merge  bool            `json:"merge,omitempty"`
	Body   json.RawMessage `json:"body,omitempty"`
}

// SearchReq is the body for POST /search against the STAC API.
type SearchReq struct {
	Collections         []string `json:"collections"`
	Limit               int      `json:"limit,omitempty"`
	ForecastRefDatetime string   `json:"forecast:reference_datetime,omitempty"`
	ForecastVariable    string   `json:"forecast:variable,omitempty"`
	ForecastPerturbed   *bool    `json:"forecast:perturbed,omitempty"`
	Cursor              string   `json:"cursor,omitempty"`
}

// FetchJSON performs a GET request to url and JSON-decodes the response body
// into a value of type T.
func FetchJSON[T any](ctx context.Context, url string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return zero, fmt.Errorf("decoding response from %s: %w", url, err)
	}
	return result, nil
}

// PostJSON performs a POST request with a JSON body to url and JSON-decodes
// the response body into a value of type T.
func PostJSON[T any](ctx context.Context, url string, body any) (T, error) {
	var zero T
	buf, err := json.Marshal(body)
	if err != nil {
		return zero, fmt.Errorf("marshalling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("POST %s: status %d", url, resp.StatusCode)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return zero, fmt.Errorf("decoding response from %s: %w", url, err)
	}
	return result, nil
}

func sortItems(items []Item) {
	slices.SortFunc(items, func(a, b Item) int {
		if c := a.Props.ReferenceDatetime.Compare(b.Props.ReferenceDatetime); c != 0 {
			return c
		}

		if c := a.Props.ValidDatetime.Compare(b.Props.ValidDatetime); c != 0 {
			return c
		}

		if c := cmp.Compare(a.Props.Variable, b.Props.Variable); c != 0 {
			return c
		}

		if !a.Props.Perturbed && b.Props.Perturbed {
			return -1
		}
		// If a is true and b is false, return 1 (b comes first)
		if a.Props.Perturbed && !b.Props.Perturbed {
			return 1
		}
		return 0 // They are equal
	})
}

// SearchItems performs a paginated POST /search request with the given parameters
// and returns all matching items. Pagination uses the cursor-based approach where
// the next link includes a body with a cursor field that is merged into subsequent
// requests.
func SearchItems(ctx context.Context, searchReq SearchReq) ([]Item, error) {
	var allItems []Item
	req := searchReq
	for {
		logg.Trace(ctx, "Searching STAC items", "collected", len(allItems), "variable", req.ForecastVariable)
		page, err := PostJSON[ItemCollection](ctx, getSearchURL(), req)
		if err != nil {
			return nil, err
		}
		allItems = append(allItems, page.Features...)

		cursor := nextCursor(page.Links)
		if cursor == "" {
			break
		}
		req.Cursor = cursor
	}

	sortItems(allItems)

	return allItems, nil
}

// nextCursor extracts the pagination cursor from the "next" link in a search
// response. The search API returns next links with method=POST, merge=true, and
// a body containing the cursor. Returns an empty string when there are no more
// pages.
func nextCursor(links []Link) string {
	for _, l := range links {
		if l.Rel != "next" {
			continue
		}
		if len(l.Body) == 0 {
			return ""
		}
		var body struct {
			Cursor string `json:"cursor"`
		}
		if err := json.Unmarshal(l.Body, &body); err != nil {
			return ""
		}
		return body.Cursor
	}
	return ""
}

// modelRunInterval is the time between consecutive ICON-CH1-EPS model runs.
const modelRunInterval = 3 * time.Hour

// maxRefTimeRetries is the number of older reference times to probe when
// the newest computed reference time has no items yet (e.g. because the
// model run is still in progress).
const maxRefTimeRetries = 8

// FetchItemsForVariables fetches the STAC collection to determine the newest
// forecast reference datetime from the temporal extent, then uses the Search API
// to retrieve items matching each requested variable for that reference time.
//
// Because the collection's temporal extent may advertise a model run that is
// still being uploaded, the function probes a single variable first. If no items
// are returned, it steps back by [modelRunInterval] (3 h) and retries up to
// [maxRefTimeRetries] times before giving up.
//
// The returned Collection can be used by callers to extract additional metadata
// (e.g. grid constants asset URLs) without a second fetch.
//
// Returns an error if the collection has no temporal extent or if any search
// request fails.
func FetchItemsForVariables(ctx context.Context, collectionURL string, variables []string, perturbed bool) ([]Item, *Collection, error) {
	coll, err := FetchJSON[Collection](ctx, collectionURL)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching STAC collection: %w", err)
	}

	refTime := coll.NewestReferenceTime()
	if refTime.IsZero() {
		return nil, &coll, nil
	}

	// Probe with the first variable to find a reference time that has data.
	probeVar := strings.ToUpper(variables[0])
	for attempt := range maxRefTimeRetries {
		candidate := refTime.Add(-time.Duration(attempt) * modelRunInterval)
		candidateStr := candidate.Format(time.RFC3339)

		probeItems, searchErr := SearchItems(ctx, SearchReq{
			Collections:         []string{CollectionID},
			ForecastRefDatetime: candidateStr,
			ForecastVariable:    probeVar,
			ForecastPerturbed:   &perturbed,
		})
		if searchErr != nil {
			return nil, nil, fmt.Errorf("probing reference time %s for variable %s: %w", candidateStr, probeVar, searchErr)
		}

		if len(probeItems) > 0 {
			refTime = candidate
			logg.Debug(ctx, "Found available reference time", "refTime", candidateStr, "attempt", attempt)
			break
		}

		logg.Debug(ctx, "No items for reference time, stepping back", "refTime", candidateStr)
		if attempt == maxRefTimeRetries-1 {
			return nil, &coll, nil
		}
	}

	refTimeStr := refTime.Format(time.RFC3339)

	// Fetch all variables for the confirmed reference time.
	var allItems []Item
	for _, v := range variables {
		items, searchErr := SearchItems(ctx, SearchReq{
			Collections:         []string{CollectionID},
			ForecastRefDatetime: refTimeStr,
			ForecastVariable:    strings.ToUpper(v),
			ForecastPerturbed:   &perturbed,
		})
		if searchErr != nil {
			return nil, nil, fmt.Errorf("searching items for variable %s: %w", v, searchErr)
		}
		logg.Debug(ctx, "Search returned items", "variable", v, "count", len(items))
		allItems = append(allItems, items...)
	}

	sortItems(allItems)

	return allItems, &coll, nil
}

// durationPattern matches ISO 8601 durations of the form P[n]DT[n]H[n]M[n]S.
var durationPattern = regexp.MustCompile(`^P(?:(\d+)D)?T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

// ParseISO8601Duration parses an ISO 8601 duration of the form P[n]DT[n]H[n]M[n]S
// and returns the equivalent [time.Duration].
func ParseISO8601Duration(s string) (time.Duration, error) {
	m := durationPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("unsupported ISO 8601 duration format: %q", s)
	}

	parseInt := func(v string) int64 {
		if v == "" {
			return 0
		}
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}

	days := parseInt(m[1])
	hours := parseInt(m[2])
	minutes := parseInt(m[3])
	seconds := parseInt(m[4])

	return time.Duration(days)*24*time.Hour +
		time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second, nil
}
