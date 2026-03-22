// Package geoadmin provides a client for the STAC API at http://data.geo.admin.ch/api/stac/v1/.
// API docs: https://data.geo.admin.ch/api/stac/static/spec/v1/api.html.
// OpenAPI spec: https://data.geo.admin.ch/api/stac/static/spec/v1/openapi.yaml.
// This package might also work with other generic STAC APIs (not tested).
// Usage:
//
//	ctx := context.Background()
//	client := geoadmin.NewClient(geoadmin.BaseURL)
//	catalog, err := client.GetCatalog(ctx)
package geoadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/paulmach/orb/geojson"
)

const BaseURL = "http://data.geo.admin.ch/api/stac/v1/"

type Client struct {
	baseURL string
}

func NewClient(baseURL string) Client {
	return Client{baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *Client) url(path string) string {
	return fmt.Sprintf("%s/%s", c.baseURL, path)
}

// getJSON fetches the given URL and decodes the JSON response into T.
func getJSON[T any](ctx context.Context, url string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// postJSON sends a POST request with a JSON body and decodes the response into T.
func postJSON[T any](ctx context.Context, url string, body any) (*T, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// nextLink returns the href of the "next" link, or empty string if absent.
func nextLink(links []Link) string {
	for _, l := range links {
		if l.Rel == "next" {
			return l.Href
		}
	}
	return ""
}

// GetCatalog fetches the root STAC catalog.
func (c *Client) GetCatalog(ctx context.Context) (*Catalog, error) {
	return getJSON[Catalog](ctx, c.url(""))
}

// CollectionsParams holds optional query parameters for GetCollections.
type CollectionsParams struct {
	// Limit controls how many collections per page (1-100, default 100).
	Limit int
	// Provider filters collections by provider name (partial, case-insensitive).
	Provider string
}

// GetCollections fetches all STAC collections, paginating through every page.
// It returns the combined list of collections from all pages.
func (c *Client) GetCollections(ctx context.Context, params CollectionsParams) ([]Collection, error) {
	u := c.url("collections")
	sep := "?"
	if params.Limit > 0 {
		u += fmt.Sprintf("%slimit=%d", sep, params.Limit)
		sep = "&"
	}
	if params.Provider != "" {
		u += fmt.Sprintf("%sprovider=%s", sep, params.Provider)
	}

	var all []Collection
	for u != "" {
		page, err := getJSON[CollectionsResponse](ctx, u)
		if err != nil {
			return nil, fmt.Errorf("fetching collections page: %w", err)
		}
		all = append(all, page.Collections...)
		u = nextLink(page.Links)
	}
	return all, nil
}

// GetCollection fetches a single STAC collection by its ID.
func (c *Client) GetCollection(ctx context.Context, collectionID string) (*Collection, error) {
	return getJSON[Collection](ctx, c.url("collections/"+collectionID))
}

// FeaturesParams holds optional query parameters for GetFeatures.
type FeaturesParams struct {
	// Limit controls how many items per page (1-100, default 100).
	Limit int
	// BBox filters items by bounding box [west, south, east, north] in WGS84.
	BBox [4]float64
	// Datetime filters items by date-time or interval (RFC 3339).
	Datetime string
}

// GetFeatures fetches all STAC items for a collection, paginating through every page.
func (c *Client) GetFeatures(ctx context.Context, collectionID string, params FeaturesParams) ([]Feature, error) {
	q := url.Values{}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.BBox != [4]float64{} {
		q.Set("bbox", fmt.Sprintf("%g,%g,%g,%g", params.BBox[0], params.BBox[1], params.BBox[2], params.BBox[3]))
	}
	if params.Datetime != "" {
		q.Set("datetime", params.Datetime)
	}

	u := c.url("collections/" + collectionID + "/items")
	if encoded := q.Encode(); encoded != "" {
		u += "?" + encoded
	}

	var all []Feature
	for u != "" {
		page, err := getJSON[ItemsResponse](ctx, u)
		if err != nil {
			return nil, fmt.Errorf("fetching items page: %w", err)
		}
		all = append(all, page.Features...)
		u = nextLink(page.Links)
	}
	return all, nil
}

// GetFeature fetches a single STAC feature by collection and feature ID.
func (c *Client) GetFeature(ctx context.Context, collectionID, featureID string) (*Feature, error) {
	return getJSON[Feature](ctx, c.url("collections/"+collectionID+"/items/"+featureID))
}

// GetAssets fetches all collection-level assets for a collection.
func (c *Client) GetAssets(ctx context.Context, collectionID string) ([]Asset, error) {
	resp, err := getJSON[AssetsResponse](ctx, c.url("collections/"+collectionID+"/assets"))
	if err != nil {
		return nil, fmt.Errorf("fetching assets: %w", err)
	}
	return resp.Assets, nil
}

// GetAsset fetches a single STAC asset by collection and asset ID.
func (c *Client) GetAsset(ctx context.Context, collectionID, assetID string) (*Asset, error) {
	return getJSON[Asset](ctx, c.url("collections/"+collectionID+"/assets/"+assetID))
}

// DownloadAsset downloads the asset object data referenced by the asset's Href.
// The caller must close the returned ReadCloser when done.
// Returns the response body and the Content-Type header.
func DownloadAsset(ctx context.Context, asset Asset) (body io.ReadCloser, contentType string, err error) {
	if asset.Href == nil {
		return nil, "", errors.New("asset has no href")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *asset.Href, nil)
	if err != nil {
		return nil, "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("unexpected status %d for %s", resp.StatusCode, *asset.Href)
	}

	return resp.Body, resp.Header.Get("Content-Type"), nil
}

// SearchParams holds optional query parameters for Search.
type SearchParams struct {
	// Limit controls how many features per page (1-100, default 100).
	Limit int
	// BBox filters by bounding box [west, south, east, north] in WGS84.
	BBox [4]float64
	// Intersects filters by a GeoJSON geometry. Mutually exclusive with BBox.
	Intersects *geojson.Geometry
	// Datetime filters by date-time or interval (RFC 3339).
	Datetime string
	// IDs filters by item IDs. When set, other filters are ignored by the server.
	IDs []string
	// Collections filters by collection IDs.
	Collections []string
}

// Search queries the STAC search endpoint, paginating through all results.
func (c *Client) Search(ctx context.Context, params SearchParams) ([]Feature, error) {
	q := url.Values{}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.BBox != [4]float64{} {
		q.Set("bbox", fmt.Sprintf("%g,%g,%g,%g", params.BBox[0], params.BBox[1], params.BBox[2], params.BBox[3]))
	}
	if params.Intersects != nil {
		b, err := json.Marshal(params.Intersects)
		if err != nil {
			return nil, fmt.Errorf("marshaling intersects geometry: %w", err)
		}
		q.Set("intersects", string(b))
	}
	if params.Datetime != "" {
		q.Set("datetime", params.Datetime)
	}
	if len(params.IDs) > 0 {
		q.Set("ids", strings.Join(params.IDs, ","))
	}
	if len(params.Collections) > 0 {
		q.Set("collections", strings.Join(params.Collections, ","))
	}

	u := c.url("search")
	if encoded := q.Encode(); encoded != "" {
		u += "?" + encoded
	}

	var all []Feature
	for u != "" {
		page, err := getJSON[ItemsResponse](ctx, u)
		if err != nil {
			return nil, fmt.Errorf("fetching search page: %w", err)
		}
		all = append(all, page.Features...)
		u = nextLink(page.Links)
	}
	return all, nil
}

// nextCursor extracts the pagination cursor from the "next" link in a POST
// search response. Returns empty string when there are no more pages.
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

// SearchPostBody is the JSON request body for POST /search.
type SearchPostBody struct {
	// Collections filters by collection IDs.
	Collections []string `json:"collections,omitempty"`
	// IDs filters by item IDs. When set, other filters are ignored by the server.
	IDs []string `json:"ids,omitempty"`
	// BBox filters by bounding box [west, south, east, north] in WGS84.
	BBox *[4]float64 `json:"bbox,omitempty"`
	// Intersects filters by a GeoJSON geometry. Mutually exclusive with BBox.
	Intersects *geojson.Geometry `json:"intersects,omitempty"`
	// Datetime filters by date-time or interval (RFC 3339).
	Datetime string `json:"datetime,omitempty"`
	// Limit controls how many features per page (1-100, default 100).
	Limit int `json:"limit,omitempty"`
	// Query defines property-level filters.
	Query map[string]any `json:"query,omitempty"`
	// Forecast extension fields.
	ForecastReferenceDatetime string `json:"forecast:reference_datetime,omitempty"`
	ForecastHorizon           string `json:"forecast:horizon,omitempty"`
	ForecastDuration          string `json:"forecast:duration,omitempty"`
	ForecastVariable          string `json:"forecast:variable,omitempty"`
	ForecastPerturbed         *bool  `json:"forecast:perturbed,omitempty"`
	// Cursor is used internally for pagination; callers should leave it empty.
	Cursor string `json:"cursor,omitempty"`
}

// SearchPost performs a paginated POST /search request and returns all matching
// features. Pagination uses the cursor-based approach where the next link's body
// contains a cursor that is merged into subsequent requests.
func (c *Client) SearchPost(ctx context.Context, body SearchPostBody) ([]Feature, error) {
	u := c.url("search")
	req := body
	var all []Feature
	for {
		page, err := postJSON[ItemsResponse](ctx, u, req)
		if err != nil {
			return nil, fmt.Errorf("fetching search page: %w", err)
		}
		all = append(all, page.Features...)

		cursor := nextCursor(page.Links)
		if cursor == "" {
			break
		}
		req.Cursor = cursor
	}
	return all, nil
}
