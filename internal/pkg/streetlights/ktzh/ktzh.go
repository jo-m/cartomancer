// Package ktzh provides a client and async job for fetching public street
// lighting operated by the Canton of Zurich from the OGD WFS endpoint
// (https://maps.zh.ch/wfs/OGDZHWFS, layer ms:ogd-0124_giszhpub_tba_str_beleuchtung_p).
package ktzh

import (
	"context"
	"fmt"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
)

const (
	baseURL  = "https://maps.zh.ch/wfs/OGDZHWFS"
	typeName = "ms:ogd-0124_giszhpub_tba_str_beleuchtung_p"

	// srsName forces the server to return geometries in WGS84. With the URN
	// form, MapServer emits CRS84 GeoJSON (lon, lat axis order).
	srsName = "urn:ogc:def:crs:EPSG::4326"

	// pageCount controls server-side pagination of GetFeature.
	pageCount = 1000
)

// DataAttribution is the user-facing data source credit for the Canton Zurich
// street lighting feed.
var DataAttribution = attribute.Attribution{
	What:       "Street Lighting (Canton Zurich)",
	Title:      "Oeffentliche Beleuchtung (OGD)",
	Author:     "Tiefbauamt Kanton Zürich",
	Source:     "https://www.zh.ch/de/politik-staat/opendata.html",
	License:    "CC0 1.0",
	LicenseURL: "https://creativecommons.org/publicdomain/zero/1.0/",
}

// Fetch retrieves all lamp point features advertised on the canton's public
// lighting layer, decoding each feature's GeoJSON payload into a [Feature].
func Fetch(ctx context.Context) ([]Feature, error) {
	c := wfs.NewClient(baseURL)
	raw, err := c.GetFeatureGeoJSON(ctx, wfs.GetFeatureParams{
		TypeNames: typeName,
		SRSName:   srsName,
		Count:     pageCount,
	})
	if err != nil {
		return nil, fmt.Errorf("get feature: %w", err)
	}

	out := make([]Feature, 0, len(raw))
	for i, r := range raw {
		f, err := decodeFeature(r)
		if err != nil {
			return nil, fmt.Errorf("decode feature %d: %w", i, err)
		}
		out = append(out, f)
	}
	return out, nil
}
