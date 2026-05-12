// Package sz provides a client and async job for fetching construction-site
// road closures from the Canton of Schwyz geoportal WFS endpoint
// (https://map.geo.sz.ch/mapserv_proxy, layer ms:ch.sz.a083a.baustellen).
package sz

import (
	"context"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
	"jo-m.ch/go/cartomancer/internal/pkg/wfs/gml"
)

const (
	baseURL  = "https://map.geo.sz.ch/mapserv_proxy"
	typeName = "ms:ch.sz.a083a.baustellen"

	// srsName forces the server to return geometries in WGS84.
	// With the EPSG URN form, WFS 2.0 mandates lat/lon axis order; the
	// shared GML decoder accounts for this.
	srsName = "urn:ogc:def:crs:EPSG::4326"

	// pageCount controls server-side pagination of GetFeature.
	pageCount = 500
)

// DataAttribution is the user-facing data source credit for the Canton of
// Schwyz construction sites feed.
var DataAttribution = attribute.Attribution{
	What:       "Road Closures (Canton Schwyz)",
	Title:      "Baustellen Kanton Schwyz",
	Author:     "Tiefbauamt Kanton Schwyz",
	Source:     "https://www.sz.ch/behoerden/verwaltung/baudepartement/tiefbauamt/baustellen.html/8756-8758-8802-9276-9290-10050",
	License:    "Opendata BY",
	LicenseURL: "https://opendata.swiss/en/terms-of-use#terms_by",
}

// Fetch retrieves all construction-site features advertised on the
// ms:ch.sz.a083a.baustellen layer, decoding each feature's GML payload
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
		f, err := decodeFeature(m.Feature, i)
		if err != nil {
			return nil, fmt.Errorf("decode feature %d: %w", i, err)
		}
		out = append(out, f)
	}
	return out, nil
}

// decodeFeature parses one WFS member into a populated [Feature]. The
// geometry element is decoded from GML into GeoJSON; remaining fields come
// from the schema-specific ms:* siblings inside the feature's InnerXML.
//
// The SZ feed does not assign meaningful gml:id values, so [Feature.SourceID]
// is derived deterministically from the feature's textual fields and (when
// available) its first geometry vertex. The index parameter is included only
// as a last-resort fallback when a feature has no geometry and no usable
// text fields, to keep IDs unique within one fetch cycle.
func decodeFeature(raw wfs.Feature, index int) (Feature, error) {
	var props featureProps
	if err := xml.Unmarshal(wrapFeature(raw.InnerXML), &props); err != nil {
		return Feature{}, fmt.Errorf("decode properties: %w", err)
	}

	geom, err := gml.DecodeGeometry(props.Geom.InnerXML)
	if err != nil {
		return Feature{}, fmt.Errorf("decode geometry: %w", err)
	}

	f := Feature{
		Lokalname:             props.Lokalname,
		Beschreibung:          props.Beschreibung,
		Behinderungsbemerkung: props.Behinderungsbemerkung,
		KontaktBauleitung:     props.KontaktBauleitung,
		KontaktTBA:            props.KontaktTBA,
		Link:                  props.Link,
		Geometry:              geom,
	}
	if t, ok := parseGermanDate(props.BaubeginnUI); ok {
		f.Baubeginn = t
	}
	if t, ok := parseGermanDate(props.Inbetriebnahme); ok {
		f.Inbetriebnahme = t
	}
	f.SourceID = sourceID(f, index)
	return f, nil
}

// sourceID builds a stable identifier for an SZ feature. The SZ WFS does
// not emit gml:id, so we hash the feature's identifying text fields plus
// its first geometry vertex. The resulting ID is stable across fetches as
// long as those fields do not change. When a feature has no text fields and
// no geometry, the within-cycle index is appended to keep the ID unique.
//
// FNV-1a is used because the hash is purely for identification, not
// authentication; a fast non-cryptographic hash is sufficient and avoids
// pulling in crypto.
func sourceID(f Feature, index int) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s|%s|%s", f.Lokalname, f.Beschreibung, f.Behinderungsbemerkung)
	if f.Geometry != nil {
		if pt, ok := firstPoint(f.Geometry); ok {
			fmt.Fprintf(h, "|%.6f|%.6f", pt.Lon(), pt.Lat())
		}
	} else {
		fmt.Fprintf(h, "|noGeom|%s", strconv.Itoa(index))
	}
	return fmt.Sprintf("sz-%016x", h.Sum64())
}

// firstPoint returns the first coordinate of a GeoJSON geometry, if it has
// one. Used to anchor the deterministic source ID to a stable geometric
// reference.
func firstPoint(g *geojson.Geometry) (orb.Point, bool) {
	switch v := g.Geometry().(type) {
	case orb.Point:
		return v, true
	case orb.LineString:
		if len(v) > 0 {
			return v[0], true
		}
	case orb.MultiPoint:
		if len(v) > 0 {
			return v[0], true
		}
	case orb.Polygon:
		if len(v) > 0 && len(v[0]) > 0 {
			return v[0][0], true
		}
	case orb.MultiLineString:
		if len(v) > 0 && len(v[0]) > 0 {
			return v[0][0], true
		}
	case orb.MultiPolygon:
		if len(v) > 0 && len(v[0]) > 0 && len(v[0][0]) > 0 {
			return v[0][0][0], true
		}
	}
	return orb.Point{}, false
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
