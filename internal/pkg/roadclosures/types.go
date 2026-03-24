package roadclosures

import "github.com/paulmach/orb/geojson"

// FindResponse is the top-level response from the MapServer find endpoint.
type FindResponse struct {
	Results []Feature `json:"results"`
}

// Feature represents a single road closure or detour feature.
type Feature struct {
	Type       string            `json:"type"` // Always "Feature".
	FeatureID  int               `json:"featureId"`
	BBox       []float64         `json:"bbox"`
	LayerBodID string            `json:"layerBodId"`
	LayerName  string            `json:"layerName"`
	ID         int               `json:"id"`
	Geometry   *geojson.Geometry `json:"geometry"`
	Properties Properties        `json:"properties"`
}

// Properties holds the metadata fields for a road closure or detour feature.
// Multilingual fields are provided in German (de), French (fr), Italian (it), and English (en).
type Properties struct {
	SperrungenType    string  `json:"sperrungen_type"` // "detour" or "closed_way".
	SperrungenTypeDe  string  `json:"sperrungen_type_de"`
	SperrungenTypeFr  string  `json:"sperrungen_type_fr"`
	SperrungenTypeIt  string  `json:"sperrungen_type_it"`
	SperrungenTypeEn  string  `json:"sperrungen_type_en"`
	Land              string  `json:"land"`
	DurationDe        string  `json:"duration_de"`
	DurationFr        string  `json:"duration_fr"`
	DurationIt        string  `json:"duration_it"`
	DurationEn        string  `json:"duration_en"`
	ReasonDe          string  `json:"reason_de"`
	ReasonFr          string  `json:"reason_fr"`
	ReasonIt          string  `json:"reason_it"`
	ReasonEn          string  `json:"reason_en"`
	TitleDe           string  `json:"title_de"`
	TitleFr           string  `json:"title_fr"`
	TitleIt           string  `json:"title_it"`
	TitleEn           string  `json:"title_en"`
	AbstractDe        string  `json:"abstract_de"`
	AbstractFr        string  `json:"abstract_fr"`
	AbstractIt        string  `json:"abstract_it"`
	AbstractEn        string  `json:"abstract_en"`
	StateValidateDe   *string `json:"state_validate_de"`
	StateValidateFr   *string `json:"state_validate_fr"`
	StateValidateIt   *string `json:"state_validate_it"`
	StateValidateEn   *string `json:"state_validate_en"`
	FileDe            *string `json:"file_de"`
	FileFr            *string `json:"file_fr"`
	FileIt            *string `json:"file_it"`
	FileEn            *string `json:"file_en"`
	ContentProviderDe string  `json:"content_provider_de"`
	ContentProviderFr string  `json:"content_provider_fr"`
	ContentProviderIt string  `json:"content_provider_it"`
	ContentProviderEn string  `json:"content_provider_en"`
	URL1LinkDe        *string `json:"url1_link_de"`
	URL1LinkFr        *string `json:"url1_link_fr"`
	URL1LinkIt        *string `json:"url1_link_it"`
	URL1LinkEn        *string `json:"url1_link_en"`
	RouteNr           *string `json:"route_nr"`
	SegmentNr         *string `json:"segment_nr"`
	Label             string  `json:"label"`
}
