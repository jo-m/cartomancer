package sg

import (
	"time"

	"github.com/paulmach/orb/geojson"
)

// Feature is one decoded ods:baustellenkoordination record from the Canton of
// St. Gallen construction-site WFS.
// All time fields are zero when missing or unparseable in the source feature;
// callers should guard with [time.Time.IsZero] before use.
type Feature struct {
	// SourceID is derived from the upstream ods:id field, prefixed with "sg-".
	SourceID string

	// Bew is the road/project label (Bewilligung name), e.g.
	// "Lütisburg Flawilerstrasse".
	Bew string

	// Zust is the responsible office, e.g. "Tiefbauamt Kanton St.Gallen".
	Zust string

	// Adresse is the civic address of the construction site.
	Adresse string

	// Beginn is the permit start date. Zero when missing or unparseable.
	Beginn time.Time

	// Ende is the permit end date. Zero when missing or unparseable.
	Ende time.Time

	// Geometry is the closure footprint in WGS84 (GeoJSON axis order: lon, lat).
	Geometry *geojson.Geometry
}

// featureProps mirrors the ods:baustellenkoordination schema for the fields
// we consume. The geometry sub-element is kept as raw XML and decoded
// separately by the [wfs/gml] package.
type featureProps struct {
	GeoShape rawXML `xml:"https://stgallen.opendatasoft.com/api/wfs/featuretype geo_shape"`
	ID       string `xml:"https://stgallen.opendatasoft.com/api/wfs/featuretype id"`
	Bew      string `xml:"https://stgallen.opendatasoft.com/api/wfs/featuretype bew"`
	Zust     string `xml:"https://stgallen.opendatasoft.com/api/wfs/featuretype zust"`
	Adresse  string `xml:"https://stgallen.opendatasoft.com/api/wfs/featuretype adresse"`
	Beginn   string `xml:"https://stgallen.opendatasoft.com/api/wfs/featuretype beginn"`
	Ende     string `xml:"https://stgallen.opendatasoft.com/api/wfs/featuretype ende"`
}

// rawXML captures the inner XML of an element so it can be decoded in a
// separate pass.
type rawXML struct {
	InnerXML []byte `xml:",innerxml"`
}
