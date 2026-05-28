// Package tg provides a client and async job for fetching construction-site
// road closures from the Canton of Thurgau ThurGIS portal, layer
// baustellen-baustelle on the chsdi3-style identify endpoint at
// https://map.geo.tg.ch/services/geofy_chsdi3/rest/services/all/MapServer/identify.
//
// The upstream endpoint returns GeoJSON with coordinates in LV95 (EPSG:2056).
// This package reprojects them to WGS84 via internal/pkg/lv95 before passing
// the features on to the shared roadclosures pipeline.
package tg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/client"
	"jo-m.ch/go/cartomancer/internal/pkg/lv95"
)

const (
	// baseURL is the chsdi3-style identify endpoint of the ThurGIS portal.
	baseURL = "https://map.geo.tg.ch/services/geofy_chsdi3/rest/services/all/MapServer/identify"

	// layer is the ThurGIS layer ID for construction sites.
	layer = "all:baustellen-baustelle"

	// Canton of Thurgau bounding box in LV95 (EPSG:2056), used as both the
	// query envelope and the simulated map extent. The endpoint requires an
	// imageDisplay + mapExtent + geometry triple but ignores their precise
	// values when tolerance is zero and the envelope already covers all data.
	cantonMinE = 2680000
	cantonMinN = 1240000
	cantonMaxE = 2775000
	cantonMaxN = 1300000
)

// DataAttribution is the user-facing data source credit for the Canton of
// Thurgau construction sites feed.
var DataAttribution = attribute.Attribution{
	What:   "Road Closures (Canton Thurgau)",
	Title:  "Baustellen Kanton Thurgau",
	Author: "Amt fuer Geoinformation, Kanton Thurgau",
	Source: "https://map.geo.tg.ch/apps/mf-geoadmin3/?lang=de&topic=verkehr&layers=baustellen-baustelle",
}

// getURL composes the URL to download all construction-site features for the
// Canton of Thurgau as GeoJSON. The chsdi3 identify action requires the
// imageDisplay, mapExtent and geometry parameters; tolerance=0 plus an
// envelope covering the whole canton selects every feature.
func getURL() string {
	envelope := fmt.Sprintf("%d,%d,%d,%d", cantonMinE, cantonMinN, cantonMaxE, cantonMaxN)
	params := url.Values{}
	params.Add("layers", layer)
	params.Add("geometry", envelope)
	params.Add("geometryType", "esriGeometryEnvelope")
	params.Add("mapExtent", envelope)
	params.Add("imageDisplay", "1024,768,96")
	params.Add("tolerance", "0")
	params.Add("returnGeometry", "true")
	params.Add("geometryFormat", "geojson")
	params.Add("lang", "de")
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// Fetch retrieves all construction-site features from the ThurGIS identify
// endpoint and reprojects their geometries from LV95 to WGS84.
func Fetch(ctx context.Context) (*IdentifyResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.New().Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result IdentifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	for i := range result.Results {
		result.Results[i].Geometry = reprojectGeometry(result.Results[i].Geometry)
	}
	return &result, nil
}

// reprojectGeometry converts every coordinate of a GeoJSON geometry from
// LV95 (EPSG:2056) to WGS84. Returns nil when the input is nil. The output
// is a freshly allocated geometry; the input is not modified.
func reprojectGeometry(g *geojson.Geometry) *geojson.Geometry {
	if g == nil {
		return nil
	}
	return geojson.NewGeometry(reprojectOrb(g.Geometry()))
}

// reprojectOrb walks an orb geometry tree, projecting every point with
// lv95.ToWGS84. Unsupported geometry types are returned unchanged; the
// upstream feed only emits Point, LineString and MultiLineString in practice.
func reprojectOrb(g orb.Geometry) orb.Geometry {
	switch v := g.(type) {
	case orb.Point:
		return reprojectPoint(v)
	case orb.MultiPoint:
		out := make(orb.MultiPoint, len(v))
		for i, p := range v {
			out[i] = reprojectPoint(p)
		}
		return out
	case orb.LineString:
		out := make(orb.LineString, len(v))
		for i, p := range v {
			out[i] = reprojectPoint(p)
		}
		return out
	case orb.MultiLineString:
		out := make(orb.MultiLineString, len(v))
		for i, ls := range v {
			line := make(orb.LineString, len(ls))
			for j, p := range ls {
				line[j] = reprojectPoint(p)
			}
			out[i] = line
		}
		return out
	case orb.Polygon:
		out := make(orb.Polygon, len(v))
		for i, ring := range v {
			r := make(orb.Ring, len(ring))
			for j, p := range ring {
				r[j] = reprojectPoint(p)
			}
			out[i] = r
		}
		return out
	case orb.MultiPolygon:
		out := make(orb.MultiPolygon, len(v))
		for i, poly := range v {
			p := make(orb.Polygon, len(poly))
			for j, ring := range poly {
				r := make(orb.Ring, len(ring))
				for k, pt := range ring {
					r[k] = reprojectPoint(pt)
				}
				p[j] = r
			}
			out[i] = p
		}
		return out
	default:
		return g
	}
}

// reprojectPoint converts a single LV95 easting/northing point to WGS84
// lon/lat. The orb.Point layout is (lon, lat) on output and (E, N) on input.
func reprojectPoint(p orb.Point) orb.Point {
	lon, lat := lv95.ToWGS84(p[0], p[1])
	return orb.Point{lon, lat}
}
