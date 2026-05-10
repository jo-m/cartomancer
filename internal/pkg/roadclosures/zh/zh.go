// Package zh provides a client and async job for fetching construction-site
// road closures from the Canton of Zurich WFS endpoint
// (https://maps.zh.ch/wfs/TbaBaustellenZHWFS, layer ms:baustellen-detailansicht).
package zh

import (
	"context"
	"encoding/xml"
	"fmt"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
)

const (
	baseURL  = "https://maps.zh.ch/wfs/TbaBaustellenZHWFS"
	typeName = "ms:baustellen-detailansicht"

	// srsName forces the server to return geometries in WGS84.
	// With the EPSG URN form, WFS 2.0 mandates lat/lon axis order; the GML
	// decoder accounts for this.
	srsName = "urn:ogc:def:crs:EPSG::4326"

	// pageCount controls server-side pagination of GetFeature.
	pageCount = 500
)

// DataAttribution is the user-facing data source credit for the Canton Zurich
// construction sites feed.
var DataAttribution = attribute.Attribution{
	What:       "Road Closures (Canton Zurich)",
	Title:      "Baustellen Kantonsstrassen Kanton Zürich",
	Author:     "Tiefbauamt Kanton Zürich",
	Source:     "https://www.zh.ch/de/politik-staat/opendata.html",
	License:    "CC-BY 4.0",
	LicenseURL: "https://creativecommons.org/licenses/by/4.0/",
}

// Fetch retrieves all construction-site features advertised on the
// ms:baustellen-detailansicht layer, decoding each feature's GML payload
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
	for _, m := range members {
		f, err := decodeFeature(m.Feature)
		if err != nil {
			return nil, fmt.Errorf("decode feature %s: %w", m.Feature.GMLID, err)
		}
		out = append(out, f)
	}
	return out, nil
}

// decodeFeature parses one WFS member into a populated [Feature].
// The geometry element is decoded from GML into GeoJSON; remaining fields
// come from the schema-specific ms:* siblings inside the feature's InnerXML.
func decodeFeature(raw wfs.Feature) (Feature, error) {
	var props featureProps
	if err := xml.Unmarshal(wrapFeature(raw.InnerXML), &props); err != nil {
		return Feature{}, fmt.Errorf("decode properties: %w", err)
	}

	geom, err := decodeGeometry(props.Geometry.InnerXML)
	if err != nil {
		return Feature{}, fmt.Errorf("decode geometry: %w", err)
	}

	f := Feature{
		GMLID:            raw.GMLID,
		Strassenname:     props.Strassenname,
		Gemeindename:     props.Gemeindename,
		Beschreibung:     props.Beschreibung,
		Verkehrsfuehrung: props.Verkehrsfuehrung,
		StatusBaustelle:  props.StatusBaustelle,
		Geometry:         geom,
	}
	if t, ok := props.DatumBaubeginn.parse(); ok {
		f.DatumBaubeginn = t
	}
	if t, ok := props.DatumBauende.parse(); ok {
		f.DatumBauende = t
	}
	return f, nil
}

// wrapFeature surrounds the raw inner XML with a synthetic root element so
// it can be parsed in one go. The wrapper declares the gml/ms namespaces
// expected by the property and geometry types.
func wrapFeature(inner []byte) []byte {
	const header = `<f xmlns:gml="http://www.opengis.net/gml/3.2" xmlns:ms="http://mapserver.gis.umn.edu/mapserver">`
	const footer = `</f>`
	buf := make([]byte, 0, len(header)+len(inner)+len(footer))
	buf = append(buf, header...)
	buf = append(buf, inner...)
	buf = append(buf, footer...)
	return buf
}
