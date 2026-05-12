package sz

import (
	"time"

	"github.com/paulmach/orb/geojson"
)

// Feature is one decoded ms:ch.sz.a083a.baustellen record.
// All time fields are zero when missing or unparseable on the source
// feature; they should be guarded with [time.Time.IsZero] before use.
type Feature struct {
	// SourceID is a deterministic identifier derived from feature content,
	// because the SZ WFS does not emit gml:id on its features.
	SourceID string

	// Lokalname is the road and section label, e.g. "2 / Gotthardstrasse, Rösslimatt, Seewen".
	Lokalname string

	// Beschreibung is a short description of the works.
	Beschreibung string

	// Behinderungsbemerkung describes how traffic is affected (lane closures, traffic lights, ...).
	Behinderungsbemerkung string

	// KontaktBauleitung is the contact info of the on-site construction lead. May be empty.
	KontaktBauleitung string

	// KontaktTBA is the contact info at the Tiefbauamt. May be empty.
	KontaktTBA string

	// Link is an optional URL to a project page.
	Link string

	// Baubeginn is the planned construction start date, parsed best-effort
	// from the human-readable baubeginn_ui field. Zero when missing or
	// unparseable.
	Baubeginn time.Time

	// Inbetriebnahme is the planned commissioning (end) date, parsed
	// best-effort from the human-readable inbetriebnahme field. Zero when
	// missing or unparseable.
	Inbetriebnahme time.Time

	// Geometry is the closure footprint in WGS84 (GeoJSON axis order: lon, lat).
	Geometry *geojson.Geometry
}

// featureProps mirrors the ms:ch.sz.a083a.baustellen schema for the
// fields we actually consume. The geometry sub-element is kept as raw XML
// and decoded separately by the [wfs/gml] package.
type featureProps struct {
	Geom                  rawXML `xml:"http://mapserver.gis.umn.edu/mapserver geom"`
	Lokalname             string `xml:"http://mapserver.gis.umn.edu/mapserver lokalname"`
	BaubeginnUI           string `xml:"http://mapserver.gis.umn.edu/mapserver baubeginn_ui"`
	Inbetriebnahme        string `xml:"http://mapserver.gis.umn.edu/mapserver inbetriebnahme"`
	Beschreibung          string `xml:"http://mapserver.gis.umn.edu/mapserver beschreibung"`
	Behinderungsbemerkung string `xml:"http://mapserver.gis.umn.edu/mapserver behinderungsbemerkung"`
	KontaktBauleitung     string `xml:"http://mapserver.gis.umn.edu/mapserver kontaktbauleitung"`
	KontaktTBA            string `xml:"http://mapserver.gis.umn.edu/mapserver kontakttba"`
	Link                  string `xml:"http://mapserver.gis.umn.edu/mapserver link"`
}

// rawXML captures the inner XML of an element, so it can be parsed in a
// separate pass.
type rawXML struct {
	InnerXML []byte `xml:",innerxml"`
}
