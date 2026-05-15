// Package sg provides a client and async job for fetching construction-site
// road closures from the Canton of St. Gallen open data portal
// (https://stgallen.opendatasoft.com, dataset baustellenkoordination).
//
// The data is published by the Amt fuer Raumentwicklung und Geoinformation
// (AREG) of Canton St. Gallen under CC BY 4.0 and is updated weekly.
// The City of St. Gallen is NOT included (separate dataset).
package sg

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
	"jo-m.ch/go/cartomancer/internal/pkg/wfs/gml"
)

const (
	baseURL  = "https://stgallen.opendatasoft.com/api/wfs"
	typeName = "ods:baustellenkoordination"

	// srsName forces the server to return geometries in WGS84.
	// With the EPSG URN form, WFS 2.0 mandates lat/lon axis order; the
	// shared GML decoder accounts for this.
	srsName = "urn:ogc:def:crs:EPSG::4326"

	// pageCount controls server-side pagination of GetFeature.
	// The dataset is small (~300 records), so one page is sufficient.
	pageCount = 500

	// odsPropNS is the XML namespace for ODS feature properties.
	odsPropNS = "https://stgallen.opendatasoft.com/api/wfs/featuretype"
)

// DataAttribution is the user-facing data source credit for the Canton of
// St. Gallen construction sites feed.
var DataAttribution = attribute.Attribution{
	What:       "Road Closures (Canton St. Gallen)",
	Title:      "Strassenbaustellen Kanton St. Gallen",
	Author:     "Amt fuer Raumentwicklung und Geoinformation (AREG), Kanton St. Gallen",
	Source:     "https://stgallen.opendatasoft.com/explore/dataset/baustellenkoordination/",
	License:    "CC BY 4.0",
	LicenseURL: "https://creativecommons.org/licenses/by/4.0/",
}

// Fetch retrieves all construction-site features advertised on the
// ods:baustellenkoordination layer, decoding each feature's GML payload
// into a [Feature].
func Fetch(ctx context.Context) ([]Feature, error) {
	c := wfs.NewClient(baseURL)
	members, err := c.GetFeature(ctx, wfs.GetFeatureParams{
		TypeNames: typeName,
		SRSName:   srsName,
		Count:     pageCount,
	})
	if err != nil {
		return nil, fmt.Errorf("get feature: %w", err)
	}

	out := make([]Feature, 0, len(members))
	for i, m := range members {
		f, err := decodeFeature(m.Feature)
		if err != nil {
			return nil, fmt.Errorf("decode feature %d: %w", i, err)
		}
		out = append(out, f)
	}
	return out, nil
}

// decodeFeature parses one WFS member into a populated [Feature]. The
// geometry element is decoded from GML into GeoJSON; remaining fields come
// from the schema-specific ods:* siblings inside the feature's InnerXML.
func decodeFeature(raw wfs.Feature) (Feature, error) {
	var props featureProps
	if err := xml.Unmarshal(wrapFeature(raw.InnerXML), &props); err != nil {
		return Feature{}, fmt.Errorf("decode properties: %w", err)
	}

	geom, err := gml.DecodeGeometry(props.GeoShape.InnerXML)
	if err != nil {
		return Feature{}, fmt.Errorf("decode geometry: %w", err)
	}

	f := Feature{
		SourceID: "sg-" + props.ID,
		Bew:      props.Bew,
		Zust:     props.Zust,
		Adresse:  props.Adresse,
		Geometry: geom,
	}
	if t, err := parseDateTime(props.Beginn); err == nil {
		f.Beginn = t
	}
	if t, err := parseDateTime(props.Ende); err == nil {
		f.Ende = t
	}
	return f, nil
}

// parseDateTime parses the ODS datetime format "2006-01-02 15:04:05-07:00"
// used by the stgallen.opendatasoft.com WFS.
func parseDateTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05-07:00", s)
}

// wrapFeature surrounds the raw inner XML with a synthetic root element so
// it can be parsed in one go. The wrapper declares the gml and ods namespaces
// expected by the property and geometry types.
func wrapFeature(inner []byte) []byte {
	const header = `<f xmlns:gml="http://www.opengis.net/gml/3.2" xmlns:ods="` + odsPropNS + `">`
	const footer = `</f>`
	buf := make([]byte, 0, len(header)+len(inner)+len(footer))
	buf = append(buf, header...)
	buf = append(buf, inner...)
	buf = append(buf, footer...)
	return buf
}
