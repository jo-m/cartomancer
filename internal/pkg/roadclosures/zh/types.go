package zh

import (
	"time"

	"github.com/paulmach/orb/geojson"
)

// Feature is one decoded ms:baustellen-detailansicht record.
// All time fields are zero when missing on the source feature; they should be
// guarded with [time.Time.IsZero] before use.
type Feature struct {
	// GMLID is the feature's gml:id attribute, e.g. "baustellen-detailansicht.2744".
	GMLID string

	// Strassenname is the street name and segment, used as the closure title.
	Strassenname string

	// Gemeindename is the municipality the closure is in.
	Gemeindename string

	// Beschreibung is a short description of the works.
	Beschreibung string

	// Verkehrsfuehrung describes how traffic is routed past the site.
	Verkehrsfuehrung string

	// StatusBaustelle is the construction status, e.g. "aktiv (Bauzeit)"
	// or "zukünftig (Bauzeit in Zukunft)".
	StatusBaustelle string

	// DatumBaubeginn is the planned construction start date. Zero when missing.
	DatumBaubeginn time.Time
	// DatumBauende is the planned construction end date. Zero when missing.
	DatumBauende time.Time

	// Geometry is the closure footprint in WGS84 (GeoJSON axis order: lon, lat).
	Geometry *geojson.Geometry
}

// featureProps mirrors the ms:baustellen-detailansicht schema for the
// fields we actually consume. The geometry sub-element is kept as raw XML
// and decoded separately (see [decodeGeometry]).
type featureProps struct {
	Geometry         rawXML       `xml:"http://mapserver.gis.umn.edu/mapserver geometry"`
	Strassenname     string       `xml:"http://mapserver.gis.umn.edu/mapserver strassenname"`
	Gemeindename     string       `xml:"http://mapserver.gis.umn.edu/mapserver gemeindename"`
	Beschreibung     string       `xml:"http://mapserver.gis.umn.edu/mapserver beschreibung"`
	Verkehrsfuehrung string       `xml:"http://mapserver.gis.umn.edu/mapserver verkehrsfuehrung"`
	StatusBaustelle  string       `xml:"http://mapserver.gis.umn.edu/mapserver status_baustelle"`
	DatumBaubeginn   gmlTimestamp `xml:"http://mapserver.gis.umn.edu/mapserver datum_baubeginn"`
	DatumBauende     gmlTimestamp `xml:"http://mapserver.gis.umn.edu/mapserver datum_bauende"`
}

// rawXML captures the inner XML of an element, so it can be parsed in a
// separate pass.
type rawXML struct {
	InnerXML []byte `xml:",innerxml"`
}

// gmlTimestamp wraps a <ms:datum_*><gml:timePosition>YYYY-MM-DDTHH:MM:SSZ</></>
// pair.
type gmlTimestamp struct {
	TimePosition string `xml:"http://www.opengis.net/gml/3.2 timePosition"`
}

// parse attempts to parse the wrapped GML time position as RFC 3339 and
// returns it together with a validity flag.
func (g gmlTimestamp) parse() (time.Time, bool) {
	if g.TimePosition == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, g.TimePosition)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
