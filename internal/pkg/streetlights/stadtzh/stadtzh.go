// Package stadtzh provides a client and async job for fetching public street
// lighting operated by the City of Zurich from the OGD WFS endpoint
// (https://www.ogd.stadt-zuerich.ch/wfs/geoportal/Oeffentliche_Beleuchtung_der_Stadt_Zuerich,
// layer ewz_brennstelle_p). The endpoint only speaks WFS 1.1.0.
package stadtzh

import (
	"context"
	"fmt"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
)

const (
	baseURL  = "https://www.ogd.stadt-zuerich.ch/wfs/geoportal/Oeffentliche_Beleuchtung_der_Stadt_Zuerich"
	typeName = "ewz_brennstelle_p"

	// srsName forces the server to return geometries in WGS84. The short
	// EPSG form is required: the URN form makes the server fail.
	srsName = "EPSG:4326"

	// pageCount controls server-side pagination of GetFeature.
	pageCount = 1000
)

// DataAttribution is the user-facing data source credit for the City of Zurich
// street lighting feed.
var DataAttribution = attribute.Attribution{
	What:       "Street Lighting (City of Zurich)",
	Title:      "Oeffentliche Beleuchtung der Stadt Zuerich",
	Author:     "Elektrizitätswerk der Stadt Zürich (ewz)",
	Source:     "https://data.stadt-zuerich.ch/dataset/geo_oeffentliche_beleuchtung_der_stadt_zuerich",
	License:    "CC0 1.0",
	LicenseURL: "https://creativecommons.org/publicdomain/zero/1.0/",
}

// Fetch retrieves all lamp point features advertised on the city's public
// lighting layer, decoding each feature's GeoJSON payload into a [Feature].
func Fetch(ctx context.Context) ([]Feature, error) {
	c := wfs.NewClientV11(baseURL)
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
