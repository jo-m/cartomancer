// Package stac provides helpers for fetching and parsing STAC (SpatioTemporal
// Asset Catalog) items from the MeteoSwiss ICON-CH1-EPS forecast API.
package stac

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"jo-m.ch/go/detour/internal/pkg/logg"
)

// Collection is a partial STAC Collection object containing only the fields
// needed to locate the parameter CSV asset.
type Collection struct {
	Assets map[string]Asset `json:"assets"`
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
	ID         string           `json:"id"`
	BBox       []float64        `json:"bbox"`
	Properties ItemProperties   `json:"properties"`
	Assets     map[string]Asset `json:"assets"`
}

// ItemProperties holds the metadata of a STAC item.
type ItemProperties struct {
	ReferenceDatetime string `json:"forecast:reference_datetime"`
	Variable          string `json:"forecast:variable"`
	Horizon           string `json:"forecast:horizon"`
	Perturbed         bool   `json:"forecast:perturbed"`
}

// Link is a STAC hyperlink.
type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
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

// FetchItemsForVariables paginates through all pages of the STAC items
// endpoint, determines the newest forecast:reference_datetime across all
// returned items, and returns those items from that run whose
// forecast:variable matches any of the requested variable names
// (case-insensitive). Both perturbed and non-perturbed items are included;
// callers filter as needed.
//
// The second return value is the reference time of the newest run (the model
// initialisation time taken directly from forecast:reference_datetime). If no
// items exist, the zero time is returned with a nil error.
//
// Note: the API does not support reliable server-side sorting, so all pages
// are fetched before the newest reference time can be determined.
func FetchItemsForVariables(ctx context.Context, baseURL string, variables []string) ([]Item, time.Time, error) {
	varSet := make(map[string]bool, len(variables))
	for _, v := range variables {
		varSet[strings.ToUpper(v)] = true
	}

	// Collect every item across all pages first, because the API does not
	// support reliable sorting by datetime.
	pageURL := fmt.Sprintf("%s/items", baseURL)
	var allItems []Item
	for pageURL != "" {
		logg.Trace(ctx, "Fetching STAC items page", "collected", len(allItems), "url", pageURL)
		page, err := FetchJSON[ItemCollection](ctx, pageURL)
		if err != nil {
			return nil, time.Time{}, err
		}
		allItems = append(allItems, page.Features...)
		pageURL = nextPageURL(page.Links)
	}

	// Determine the newest forecast reference time (model initialisation time).
	var newestRefTime time.Time
	for _, item := range allItems {
		if item.Properties.ReferenceDatetime == "" {
			continue
		}
		refTime, err := time.Parse(time.RFC3339, item.Properties.ReferenceDatetime)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("parse forecast:reference_datetime %q: %w", item.Properties.ReferenceDatetime, err)
		}
		if refTime.After(newestRefTime) {
			newestRefTime = refTime
		}
	}

	if newestRefTime.IsZero() {
		return nil, time.Time{}, nil
	}

	// Return items belonging to the newest run that match the requested variables.
	var result []Item
	for _, item := range allItems {
		if item.Properties.ReferenceDatetime == "" || item.Properties.Horizon == "" {
			continue
		}
		refTime, _ := time.Parse(time.RFC3339, item.Properties.ReferenceDatetime)
		if !refTime.Equal(newestRefTime) {
			continue
		}
		if !varSet[strings.ToUpper(item.Properties.Variable)] {
			continue
		}
		result = append(result, item)
	}

	return result, newestRefTime, nil
}

// nextPageURL returns the href of the "next" pagination link, or an empty
// string when there are no more pages.
func nextPageURL(links []Link) string {
	for _, l := range links {
		if l.Rel == "next" {
			return l.Href
		}
	}
	return ""
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
