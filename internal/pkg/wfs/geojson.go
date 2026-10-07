package wfs

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

	"jo-m.ch/go/cartomancer/internal/pkg/client"
)

// maxGeoJSONPages bounds the number of pages [Client.GetFeatureGeoJSON]
// fetches, so that a server that ignores startIndex cannot cause an endless
// loop.
const maxGeoJSONPages = 10000

// GeoJSONFeature is one feature returned by [Client.GetFeatureGeoJSON].
// The schema-specific attributes are kept as raw JSON so callers can decode
// their own types, mirroring how [Feature] keeps raw XML for GML responses.
type GeoJSONFeature struct {
	// ID is the GeoJSON feature's "id" member. Empty when the server omits it.
	ID string
	// Properties is the verbatim "properties" object of the feature.
	Properties json.RawMessage
	// Geometry is the decoded feature geometry in WGS84 (GeoJSON axis order:
	// lon, lat). Nil when the feature has no geometry.
	Geometry *geojson.Geometry
}

// geoJSONResponse is the subset of a GeoJSON FeatureCollection response
// decoded by [getGeoJSONPage].
type geoJSONResponse struct {
	Features []geoJSONFeature `json:"features"`
}

// geoJSONFeature is one raw feature inside a [geoJSONResponse].
type geoJSONFeature struct {
	ID         string          `json:"id"`
	Properties json.RawMessage `json:"properties"`
	Geometry   json.RawMessage `json:"geometry"`
}

// geoJSONOutputFormat returns the outputFormat value the client's WFS version
// expects for GeoJSON responses. MapServer and QGIS Server advertise differing
// MIME types and each rejects the other's.
func (c *Client) geoJSONOutputFormat() string {
	if c.version == Version11 {
		return "application/vnd.geo+json"
	}
	return "application/json"
}

// typeNameParam returns the layer name query parameter name for the client's
// WFS version ("typeNames" for 2.0, "typeName" for 1.1).
func (c *Client) typeNameParam() string {
	if c.version == Version11 {
		return "typeName"
	}
	return "typeNames"
}

// countParam returns the page size query parameter name for the client's WFS
// version ("count" for 2.0, "maxFeatures" for 1.1).
func (c *Client) countParam() string {
	if c.version == Version11 {
		return "maxFeatures"
	}
	return "count"
}

// GetFeatureGeoJSON fetches every feature of a layer as GeoJSON, using the
// query parameter names and output format of the client's WFS version. Pages
// are requested with an increasing startIndex until the server returns an
// empty page; the returned slice preserves server order.
//
// Count must be positive and is used as the page size. SRSName must be given
// in the form the service expects for its WFS version (see
// [GetFeatureParams]). GeoJSON coordinates are always decoded as WGS84
// lon/lat.
//
// Returns an error if a request fails or a response cannot be decoded. When
// the server responds with an OWS exception report, the returned error is an
// [*ExceptionReport].
func (c *Client) GetFeatureGeoJSON(ctx context.Context, params GetFeatureParams) ([]GeoJSONFeature, error) {
	if params.TypeNames == "" {
		return nil, errors.New("wfs: GetFeatureGeoJSON requires TypeNames")
	}
	if params.Count <= 0 {
		return nil, errors.New("wfs: GetFeatureGeoJSON requires a positive Count")
	}

	var all []GeoJSONFeature
	for page := 0; page < maxGeoJSONPages; page++ {
		q := url.Values{}
		q.Set("request", "GetFeature")
		q.Set(c.typeNameParam(), params.TypeNames)
		q.Set(c.countParam(), strconv.Itoa(params.Count))
		q.Set("startIndex", strconv.Itoa(len(all)))
		q.Set("outputFormat", c.geoJSONOutputFormat())
		if params.SRSName != "" {
			q.Set("srsName", params.SRSName)
		}

		features, err := getGeoJSONPage(ctx, c.requestURL(q))
		if err != nil {
			return nil, fmt.Errorf("get feature page at offset %d: %w", len(all), err)
		}
		if len(features) == 0 {
			return all, nil
		}
		all = append(all, features...)
	}
	return nil, fmt.Errorf("wfs: GetFeatureGeoJSON exceeded %d pages", maxGeoJSONPages)
}

// getGeoJSONPage fetches and decodes one GeoJSON GetFeature page.
func getGeoJSONPage(ctx context.Context, reqURL string) ([]GeoJSONFeature, error) {
	body, err := fetchJSONBody(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var resp geoJSONResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	out := make([]GeoJSONFeature, 0, len(resp.Features))
	for _, f := range resp.Features {
		feat := GeoJSONFeature{ID: f.ID, Properties: f.Properties}
		if len(f.Geometry) > 0 && !bytes.Equal(bytes.TrimSpace(f.Geometry), []byte("null")) {
			geom, err := geojson.UnmarshalGeometry(f.Geometry)
			if err != nil {
				return nil, fmt.Errorf("decode geometry: %w", err)
			}
			feat.Geometry = geom
		}
		out = append(out, feat)
	}
	return out, nil
}

// fetchJSONBody executes a GET request and returns the response body, which
// must be a JSON object. An OWS exception report is returned as an
// [*ExceptionReport] error. Unlike [fetchXML], the body is retained in the
// error for non-OK statuses and non-JSON payloads, since it is often the only
// diagnostic for a malformed request.
func fetchJSONBody(ctx context.Context, reqURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.New().Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if exc := parseException(body); exc != nil {
		return nil, exc
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s: %s", resp.StatusCode, reqURL, bodySnippet(body))
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("unexpected non-JSON response for %s: %s", reqURL, bodySnippet(body))
	}
	return body, nil
}

// bodySnippet returns a short, whitespace-normalised prefix of body for
// inclusion in error messages.
func bodySnippet(body []byte) string {
	const maxLen = 200
	s := strings.Join(strings.Fields(string(body)), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
