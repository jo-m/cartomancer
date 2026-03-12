package forecast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
type stacItem struct {
	ID         string               `json:"id"`
	Properties stacItemProperties   `json:"properties"`
	Assets     map[string]stacAsset `json:"assets"`
}

// stacItemProperties holds the datetime metadata of a STAC item.
type stacItemProperties struct {
	Datetime string `json:"datetime"`
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
// all items from the newest forecast datetime whose assets match any of the
// requested variable names.
//
// It also returns the forecast reference datetime string (e.g., "2026-03-10T18:00:00Z")
// for use in error messages and logging.
func fetchItemsForVariables(ctx context.Context, variables []string) ([]stacItem, string, error) {
	varSet := make(map[string]struct{}, len(variables))
	for _, v := range variables {
		varSet[strings.ToLower(v)] = struct{}{}
	}

	pageURL := fmt.Sprintf("%s/items?limit=%d&sortby=-datetime", collectionBaseURL, itemsPageSize)
	var newestDatetime string
	var result []stacItem

	for pageURL != "" {
		page, err := fetchJSON[stacItemCollection](ctx, pageURL)
		if err != nil {
			return nil, newestDatetime, err
		}

		for _, item := range page.Features {
			dt := item.Properties.Datetime
			if newestDatetime == "" {
				newestDatetime = dt
			}
			// Stop when we encounter items from an older forecast run.
			if dt != newestDatetime {
				return result, newestDatetime, nil
			}
			for assetKey := range item.Assets {
				if assetMatchesAny(assetKey, varSet) {
					result = append(result, item)
					break
				}
			}
		}

		pageURL = nextPageURL(page.Links)
	}

	return result, newestDatetime, nil
}

// assetMatchesAny reports whether assetKey contains any of the requested
// variable names. Asset keys follow the pattern
// "icon-ch1-eps-{datetime}-0-{variable}-{type}.grib2".
func assetMatchesAny(assetKey string, varSet map[string]struct{}) bool {
	lower := strings.ToLower(assetKey)
	for v := range varSet {
		if strings.Contains(lower, "-0-"+v+"-") {
			return true
		}
	}
	return false
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
