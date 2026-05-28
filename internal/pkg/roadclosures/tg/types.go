package tg

import "github.com/paulmach/orb/geojson"

// IdentifyResponse is the top-level JSON envelope returned by the chsdi3
// identify endpoint. Only the fields consumed by this package are declared.
type IdentifyResponse struct {
	Results []Feature `json:"results"`
	Success bool      `json:"success"`
}

// Feature is a single construction-site record from the
// baustellen-baustelle layer. Geometry coordinates are LV95 (EPSG:2056)
// as delivered by the server and are reprojected to WGS84 inside [Fetch].
type Feature struct {
	Type       string            `json:"type"`
	FeatureID  string            `json:"featureId"`
	ID         string            `json:"id"`
	LayerBodID string            `json:"layerBodId"`
	LayerName  string            `json:"layerName"`
	Geometry   *geojson.Geometry `json:"geometry"`
	Properties Properties        `json:"properties"`
}

// Properties holds the schema-specific fields of one ThurGIS construction
// site. Field names mirror the upstream attribute table; numeric fields
// arrive as JSON strings, so they are kept as strings here.
type Properties struct {
	// ObjectID is the upstream primary key, as a decimal string.
	ObjectID string `json:"objectid"`

	// Achsnummer is the road axis identifier (e.g. "H451").
	Achsnummer string `json:"achsnummer"`

	// Projektnummer is the cantonal project number.
	Projektnummer string `json:"projektnummer"`

	// Projektbezeichnung is the human-readable project name.
	Projektbezeichnung string `json:"projektbezeichnung"`

	// Taetigkeitsbeschrieb is a free-text description of the works.
	Taetigkeitsbeschrieb string `json:"taetigkeitsbeschrieb"`

	// TerminVon is the planned start date in YYYY-MM-DD form.
	TerminVon string `json:"terminvon"`

	// TerminBis is the planned end date in YYYY-MM-DD form.
	TerminBis string `json:"terminbis"`

	// StatusBez is the upstream status label.
	StatusBez string `json:"status_bez"`

	// StatusYear is the year associated with the current status.
	StatusYear string `json:"status_year"`
}
