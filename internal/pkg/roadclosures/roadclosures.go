// Package roadclosures provides types and a client for fetching bike road
// closures and detours from the geo.admin.ch MapServer find endpoint.
package roadclosures

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
)

const (
	baseURL = "https://api3.geo.admin.ch/rest/services/ech/MapServer/find"
	layer   = "ch.astra.veloland-sperrungen_umleitungen"
)

// getURL composes the URL to download the road closure features.
// See https://docs.geo.admin.ch/access-data/find-features.html for docs.
func getURL() string {
	params := url.Values{}
	params.Add("layer", layer)
	// searchField is required so we search for SchweizMobil in content_provider_de, which always matches.
	params.Add("searchField", "content_provider_de")
	params.Add("searchText", "SchweizMobil")
	// We want GeoJSON.
	params.Add("geometryFormat", "geojson")
	params.Add("returnGeometry", "true")
	// Spatial Reference (WGS 84 / GPS coordinates).
	params.Add("sr", "4326")
	// Language for the response labels.
	params.Add("lang", "en")

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// DataAttribution is the TASL attribution for MeteoSwiss ICON-CH1-EPS forecast data.
// Verified by TestOnlineStacLicense.
var DataAttribution = attribute.Attribution{
	What:       "Road Closures (Switzerland)",
	Title:      "Closures / Diversions \"Cycling in Switzerland\"",
	Author:     "Federal Roads Office, Canton, SwitzerlandMobility Foundation",
	Source:     "https://schweizmobil.info/de",
	License:    "GeoIV, Art. 21",
	LicenseURL: "https://schweizmobil.ch/en/copyright#copyright-2",
}

// Fetch retrieves all bike road closure and detour features from the
// geo.admin.ch MapServer find endpoint.
func Fetch(ctx context.Context) (*FindResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getURL(), nil)
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
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result FindResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}
