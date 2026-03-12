package forecast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// stacCollection is a partial STAC Collection object containing only the fields
// needed to locate the parameter CSV asset.
type stacCollection struct {
	Assets map[string]stacAsset `json:"assets"`
}

// stacAsset is a STAC Asset with a download URL and content type.
type stacAsset struct {
	Href  string `json:"href"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// stacItemCollection is a GeoJSON FeatureCollection of STAC Items with
// pagination links.
type stacItemCollection struct {
	Features []stacItem `json:"features"`
	Links    []stacLink `json:"links"`
}

// stacItem is a partial STAC Item (GeoJSON Feature) carrying forecast assets.
// BBox follows the GeoJSON convention: [min_lon, min_lat, max_lon, max_lat].
type stacItem struct {
	ID         string               `json:"id"`
	BBox       []float64            `json:"bbox"`
	Properties stacItemProperties   `json:"properties"`
	Assets     map[string]stacAsset `json:"assets"`
}

// stacItemProperties holds the metadata of a STAC item.
type stacItemProperties struct {
	Datetime  string `json:"datetime"`
	Variable  string `json:"forecast:variable"`
	Horizon   string `json:"forecast:horizon"`
	Perturbed bool   `json:"forecast:perturbed"`
}

// stacLink is a STAC hyperlink.
type stacLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// fetchJSON performs a GET request to url and JSON-decodes the response body
// into a value of type T.
func fetchJSON[T any](ctx context.Context, url string) (T, error) {
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

// fetchItemsForVariables paginates through the STAC items endpoint and returns
// all items from the newest forecast run whose forecast:variable matches any of
// the requested variable names (case-insensitive). Both perturbed and
// non-perturbed items are included; callers filter as needed.
//
// The second return value is the reference time of the newest run (i.e. the
// model initialisation time, computed as valid_time - horizon). If no items
// exist, the zero time is returned with a nil error.
func fetchItemsForVariables(ctx context.Context, variables []string) ([]stacItem, time.Time, error) {
	varSet := make(map[string]bool, len(variables))
	for _, v := range variables {
		varSet[strings.ToUpper(v)] = true
	}

	pageURL := fmt.Sprintf("%s/items?limit=%d&sortby=-datetime", collectionBaseURL, itemsPageSize)
	var newestRefTime time.Time
	foundFirst := false
	var result []stacItem
	done := false

	for pageURL != "" && !done {
		page, err := fetchJSON[stacItemCollection](ctx, pageURL)
		if err != nil {
			return nil, time.Time{}, err
		}

		for _, item := range page.Features {
			if item.Properties.Datetime == "" || item.Properties.Horizon == "" {
				continue
			}

			validTime, parseErr := time.Parse(time.RFC3339, item.Properties.Datetime)
			if parseErr != nil {
				return nil, time.Time{}, fmt.Errorf("parse item datetime %q: %w", item.Properties.Datetime, parseErr)
			}
			horizon, parseErr := parseISO8601Duration(item.Properties.Horizon)
			if parseErr != nil {
				return nil, time.Time{}, fmt.Errorf("parse item horizon %q: %w", item.Properties.Horizon, parseErr)
			}

			// Reference time is the model initialisation time; items are sorted by
			// valid time descending, so we must compute it to identify run boundaries.
			refTime := validTime.Add(-horizon)
			if !foundFirst {
				newestRefTime = refTime
				foundFirst = true
			}
			// Stop as soon as we cross into an older forecast run.
			if !refTime.Equal(newestRefTime) {
				done = true
				break
			}
			if !varSet[strings.ToUpper(item.Properties.Variable)] {
				continue
			}
			result = append(result, item)
		}

		if !done {
			pageURL = nextPageURL(page.Links)
		}
	}

	if !foundFirst {
		return nil, time.Time{}, nil
	}

	return result, newestRefTime, nil
}

// nextPageURL returns the href of the "next" pagination link, or an empty
// string when there are no more pages.
func nextPageURL(links []stacLink) string {
	for _, l := range links {
		if l.Rel == "next" {
			return l.Href
		}
	}
	return ""
}

// durationPattern matches ISO 8601 durations of the form P[n]DT[n]H[n]M[n]S.
var durationPattern = regexp.MustCompile(`^P(?:(\d+)D)?T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

// parseISO8601Duration parses an ISO 8601 duration of the form P[n]DT[n]H[n]M[n]S
// and returns the equivalent [time.Duration].
func parseISO8601Duration(s string) (time.Duration, error) {
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
