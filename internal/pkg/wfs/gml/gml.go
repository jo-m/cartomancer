// Package gml decodes GML 3.2 geometry payloads (as returned inside WFS
// 2.0 GetFeature responses) into GeoJSON geometries.
//
// The decoder operates on the raw inner XML of a property element (for
// example ms:geometry or ms:geom). It locates the first element in the
// GML namespace and parses it. The output is always (lon, lat) GeoJSON,
// regardless of the source CRS's native axis order: when the source uses
// the EPSG URN form for EPSG:4326 the decoder swaps lat/lon coordinates;
// for projected CRSes the raw (x, y) order is kept.
//
// Supported elements: gml:Point, gml:LineString, gml:Polygon and their
// gml:MultiPoint / gml:MultiCurve / gml:MultiSurface containers.
package gml

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

// gmlNS is the GML 3.2 namespace.
const gmlNS = "http://www.opengis.net/gml/3.2"

// wrapGML surrounds raw inner XML with a synthetic root that declares the
// gml namespace, so a standalone xml.Decoder resolves gml: prefixes correctly
// even when the prefix was declared on a containing element in the source.
func wrapGML(inner []byte) string {
	var b strings.Builder
	b.Grow(len(inner) + 64)
	b.WriteString(`<g xmlns:gml="`)
	b.WriteString(gmlNS)
	b.WriteString(`">`)
	b.Write(inner)
	b.WriteString(`</g>`)
	return b.String()
}

// DecodeGeometry turns the inner XML of a property element (for example
// ms:geometry or ms:geom) into a GeoJSON geometry in WGS84 (lon, lat) axis
// order. The input may include the surrounding property element, or be its
// bare inner contents; in either case the decoder finds the first element
// in the GML namespace and decodes it.
//
// The endpoint emits coordinates in the order advertised by the element's
// srsName attribute; when that attribute is the EPSG URN
// urn:ogc:def:crs:EPSG::4326 the order is lat/lon and the decoder swaps
// them on the way out. For projected CRSes such as Swiss LV95
// (urn:ogc:def:crs:EPSG::2056) the raw (x, y) order is kept.
//
// Returns (nil, nil) when inner is empty, so callers can pass through
// missing geometries without special-casing them.
func DecodeGeometry(inner []byte) (*geojson.Geometry, error) {
	if len(inner) == 0 {
		return nil, nil
	}

	dec := xml.NewDecoder(strings.NewReader(wrapGML(inner)))
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("scan geometry: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		// Skip wrapping elements (e.g. a synthetic <g> root that only
		// declares namespaces, or ms:geometry itself) until the first
		// element in the GML namespace.
		if start.Name.Space != gmlNS {
			continue
		}
		return decodeGMLElement(dec, start)
	}
}

// decodeGMLElement dispatches on the local name of a GML element.
// The decoder is positioned just past the StartElement.
func decodeGMLElement(dec *xml.Decoder, start xml.StartElement) (*geojson.Geometry, error) {
	swap := axisSwapForSRS(srsAttr(start))

	switch start.Name.Local {
	case "Point":
		p, err := decodePoint(dec, start, swap)
		if err != nil {
			return nil, err
		}
		return geojson.NewGeometry(p), nil
	case "LineString":
		ls, err := decodeLineString(dec, start, swap)
		if err != nil {
			return nil, err
		}
		return geojson.NewGeometry(ls), nil
	case "Polygon":
		poly, err := decodePolygon(dec, start, swap)
		if err != nil {
			return nil, err
		}
		return geojson.NewGeometry(poly), nil
	case "MultiPoint":
		mp, err := decodeMultiPoint(dec, start, swap)
		if err != nil {
			return nil, err
		}
		return geojson.NewGeometry(mp), nil
	case "MultiCurve", "MultiLineString":
		mls, err := decodeMultiCurve(dec, start, swap)
		if err != nil {
			return nil, err
		}
		return geojson.NewGeometry(mls), nil
	case "MultiSurface", "MultiPolygon":
		mp, err := decodeMultiSurface(dec, start, swap)
		if err != nil {
			return nil, err
		}
		return geojson.NewGeometry(mp), nil
	default:
		return nil, fmt.Errorf("unsupported GML element %s", start.Name.Local)
	}
}

// srsAttr returns the srsName attribute on a GML element, if any.
func srsAttr(start xml.StartElement) string {
	for _, a := range start.Attr {
		if a.Name.Local == "srsName" {
			return a.Value
		}
	}
	return ""
}

// axisSwapForSRS reports whether the first two coordinates in a GML pos /
// posList are encoded as (lat, lon) and need to be swapped to GeoJSON's
// (lon, lat). Per WFS 2.0 + GML 3.2, the EPSG URN form mandates the CRS's
// native axis order, which for EPSG:4326 is lat/lon.
func axisSwapForSRS(srs string) bool {
	return srs == "urn:ogc:def:crs:EPSG::4326" || srs == "urn:x-ogc:def:crs:EPSG::4326"
}

// decodePoint reads a <gml:Point> element (decoder positioned just past the
// StartElement), expecting a single <gml:pos> child.
func decodePoint(dec *xml.Decoder, start xml.StartElement, swap bool) (orb.Point, error) {
	var pos string
	err := walk(dec, start, func(name xml.Name, body string) error {
		if name.Space == gmlNS && name.Local == "pos" {
			pos = body
		}
		return nil
	})
	if err != nil {
		return orb.Point{}, err
	}
	pts, err := parsePosList(pos, swap)
	if err != nil {
		return orb.Point{}, err
	}
	if len(pts) == 0 {
		return orb.Point{}, errors.New("empty gml:pos")
	}
	return pts[0], nil
}

// decodeLineString reads a <gml:LineString> element with a <gml:posList> child.
func decodeLineString(dec *xml.Decoder, start xml.StartElement, swap bool) (orb.LineString, error) {
	var list string
	err := walk(dec, start, func(name xml.Name, body string) error {
		if name.Space == gmlNS && name.Local == "posList" {
			list = body
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	pts, err := parsePosList(list, swap)
	if err != nil {
		return nil, err
	}
	if len(pts) < 2 {
		return nil, fmt.Errorf("gml:LineString needs at least 2 points, got %d", len(pts))
	}
	return orb.LineString(pts), nil
}

// decodePolygon reads a <gml:Polygon> element: one exterior LinearRing and
// any number of interior rings.
func decodePolygon(dec *xml.Decoder, start xml.StartElement, swap bool) (orb.Polygon, error) {
	var poly orb.Polygon
	err := walkElems(dec, start, func(d *xml.Decoder, s xml.StartElement) error {
		if s.Name.Space != gmlNS {
			return d.Skip()
		}
		switch s.Name.Local {
		case "exterior", "interior":
			ring, err := decodeLinearRing(d, s, swap)
			if err != nil {
				return err
			}
			poly = append(poly, ring)
			return nil
		default:
			return d.Skip()
		}
	})
	if err != nil {
		return nil, err
	}
	if len(poly) == 0 {
		return nil, errors.New("gml:Polygon has no rings")
	}
	return poly, nil
}

// decodeLinearRing reads a <gml:exterior>/<gml:interior> wrapper and the
// LinearRing inside it.
func decodeLinearRing(dec *xml.Decoder, start xml.StartElement, swap bool) (orb.Ring, error) {
	var ring orb.Ring
	err := walkElems(dec, start, func(d *xml.Decoder, s xml.StartElement) error {
		if s.Name.Space == gmlNS && s.Name.Local == "LinearRing" {
			var list string
			err := walk(d, s, func(name xml.Name, body string) error {
				if name.Space == gmlNS && name.Local == "posList" {
					list = body
				}
				return nil
			})
			if err != nil {
				return err
			}
			pts, err := parsePosList(list, swap)
			if err != nil {
				return err
			}
			ring = orb.Ring(pts)
			return nil
		}
		return d.Skip()
	})
	if err != nil {
		return nil, err
	}
	if len(ring) < 4 {
		return nil, fmt.Errorf("gml:LinearRing needs at least 4 points, got %d", len(ring))
	}
	return ring, nil
}

// decodeMultiPoint reads a gml:MultiPoint with gml:pointMember/gml:Point children.
func decodeMultiPoint(dec *xml.Decoder, start xml.StartElement, swap bool) (orb.MultiPoint, error) {
	var mp orb.MultiPoint
	err := walkElems(dec, start, func(d *xml.Decoder, s xml.StartElement) error {
		if s.Name.Space == gmlNS && (s.Name.Local == "pointMember" || s.Name.Local == "pointMembers") {
			return walkElems(d, s, func(d2 *xml.Decoder, s2 xml.StartElement) error {
				if s2.Name.Space == gmlNS && s2.Name.Local == "Point" {
					pt, err := decodePoint(d2, s2, swap)
					if err != nil {
						return err
					}
					mp = append(mp, pt)
					return nil
				}
				return d2.Skip()
			})
		}
		return d.Skip()
	})
	if err != nil {
		return nil, err
	}
	if len(mp) == 0 {
		return nil, errors.New("gml:MultiPoint has no points")
	}
	return mp, nil
}

// decodeMultiCurve reads a gml:MultiCurve with gml:curveMember/gml:LineString children.
func decodeMultiCurve(dec *xml.Decoder, start xml.StartElement, swap bool) (orb.MultiLineString, error) {
	var mls orb.MultiLineString
	err := walkElems(dec, start, func(d *xml.Decoder, s xml.StartElement) error {
		if s.Name.Space == gmlNS && (s.Name.Local == "curveMember" || s.Name.Local == "curveMembers" || s.Name.Local == "lineStringMember") {
			return walkElems(d, s, func(d2 *xml.Decoder, s2 xml.StartElement) error {
				if s2.Name.Space == gmlNS && s2.Name.Local == "LineString" {
					ls, err := decodeLineString(d2, s2, swap)
					if err != nil {
						return err
					}
					mls = append(mls, ls)
					return nil
				}
				return d2.Skip()
			})
		}
		return d.Skip()
	})
	if err != nil {
		return nil, err
	}
	if len(mls) == 0 {
		return nil, errors.New("gml:MultiCurve has no line strings")
	}
	return mls, nil
}

// decodeMultiSurface reads a gml:MultiSurface with gml:surfaceMember/gml:Polygon children.
func decodeMultiSurface(dec *xml.Decoder, start xml.StartElement, swap bool) (orb.MultiPolygon, error) {
	var mp orb.MultiPolygon
	err := walkElems(dec, start, func(d *xml.Decoder, s xml.StartElement) error {
		if s.Name.Space == gmlNS && (s.Name.Local == "surfaceMember" || s.Name.Local == "surfaceMembers" || s.Name.Local == "polygonMember") {
			return walkElems(d, s, func(d2 *xml.Decoder, s2 xml.StartElement) error {
				if s2.Name.Space == gmlNS && s2.Name.Local == "Polygon" {
					poly, err := decodePolygon(d2, s2, swap)
					if err != nil {
						return err
					}
					mp = append(mp, poly)
					return nil
				}
				return d2.Skip()
			})
		}
		return d.Skip()
	})
	if err != nil {
		return nil, err
	}
	if len(mp) == 0 {
		return nil, errors.New("gml:MultiSurface has no polygons")
	}
	return mp, nil
}

// parsePosList parses a whitespace-separated coordinate list (2 numbers per
// point) into orb.Points in lon/lat order. When swap is true the input is
// treated as lat/lon and reordered.
func parsePosList(s string, swap bool) ([]orb.Point, error) {
	fields := strings.Fields(s)
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("posList has odd number of coordinates: %d", len(fields))
	}
	pts := make([]orb.Point, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		a, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return nil, fmt.Errorf("parse coord %q: %w", fields[i], err)
		}
		b, err := strconv.ParseFloat(fields[i+1], 64)
		if err != nil {
			return nil, fmt.Errorf("parse coord %q: %w", fields[i+1], err)
		}
		if swap {
			pts = append(pts, orb.Point{b, a})
		} else {
			pts = append(pts, orb.Point{a, b})
		}
	}
	return pts, nil
}

// walk visits each direct child element of start, calling cb with the child's
// name and accumulated chardata body.
func walk(dec *xml.Decoder, start xml.StartElement, cb func(name xml.Name, body string) error) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			var body strings.Builder
			if err := collectText(dec, &body); err != nil {
				return err
			}
			if err := cb(t.Name, body.String()); err != nil {
				return err
			}
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
}

// walkElems visits each direct child element of start, calling cb with the
// decoder positioned just past that child's StartElement. cb is responsible
// for either consuming the child or calling dec.Skip().
func walkElems(dec *xml.Decoder, start xml.StartElement, cb func(*xml.Decoder, xml.StartElement) error) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if err := cb(dec, t); err != nil {
				return err
			}
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
}

// collectText accumulates all chardata directly under the element the
// decoder is currently inside, until its matching EndElement is consumed.
// Nested elements are skipped over (their text is not collected).
func collectText(dec *xml.Decoder, out *strings.Builder) error {
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			if depth == 1 {
				out.Write(t)
			}
		}
	}
	return nil
}
