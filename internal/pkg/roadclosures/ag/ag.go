// Package ag provides a client and async job for fetching construction-site
// road closures from the Canton of Aargau ArcGIS REST MapServer
// (service ATB/Baustellen_online, layer 0 "Baustellen aktuell").
//
// The data is published by the Abteilung Tiefbau (ATB) of Canton Aargau
// and exposed publicly through the Baustellen webappviewer at
// https://arcgis.geo.ag.ch/portal/apps/webappviewer/index.html?id=767ea15935ad49ea8ff639465704c488.
package ag

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/client"
)

const (
	// baseURL is the AG ArcGIS REST query endpoint for the Baustellen layer.
	baseURL = "https://arcgis.geo.ag.ch/server/rest/services/ATB/Baustellen_online/MapServer/0/query"

	// whereActive selects sites that have not yet ended; the upstream
	// dataset is small enough that no further server-side filtering is
	// required to keep payloads reasonable.
	whereActive = "tDate >= CURRENT_TIMESTAMP"
)

// DataAttribution is the user-facing data source credit for the Canton of
// Aargau construction sites feed.
var DataAttribution = attribute.Attribution{
	What:   "Road Closures (Canton Aargau)",
	Title:  "Baustellen Kanton Aargau",
	Author: "Abteilung Tiefbau (ATB), Kanton Aargau",
	Source: "https://arcgis.geo.ag.ch/portal/apps/webappviewer/index.html?id=767ea15935ad49ea8ff639465704c488",
}

// getURL composes the URL to download all currently or future-active
// construction-site features as GeoJSON in WGS84.
func getURL() string {
	params := url.Values{}
	params.Add("where", whereActive)
	params.Add("outFields", "*")
	params.Add("outSR", "4326")
	params.Add("returnGeometry", "true")
	params.Add("f", "geojson")
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// Fetch retrieves all current and future construction-site features from
// the AG ArcGIS MapServer query endpoint, decoded as a [FeatureCollection].
func Fetch(ctx context.Context) (*FeatureCollection, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/geo+json, application/json")

	resp, err := client.New().Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var fc FeatureCollection
	if err := json.NewDecoder(resp.Body).Decode(&fc); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &fc, nil
}
