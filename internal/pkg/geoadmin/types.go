package geoadmin

import (
	"encoding/json"
	"time"

	"github.com/paulmach/orb/geojson"
)

// Link represents the STAC link object.
// For POST-based pagination, Method is "POST", Merge is true, and Body carries
// the cursor to merge into the next request.
type Link struct {
	Href     string          `json:"href"`
	Rel      string          `json:"rel"`
	Type     *string         `json:"type,omitempty"`
	Title    *string         `json:"title,omitempty"`
	Method   *string         `json:"method,omitempty"`
	Hreflang *string         `json:"hreflang,omitempty"`
	Merge    bool            `json:"merge,omitempty"`
	Body     json.RawMessage `json:"body,omitempty"`
}

// Provider represents a STAC collection provider.
type Provider struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	URL         *string  `json:"url,omitempty"`
}

// Extent represents the spatial and temporal extent of a collection.
type Extent struct {
	Spatial  SpatialExtent  `json:"spatial"`
	Temporal TemporalExtent `json:"temporal"`
}

// SpatialExtent represents the spatial extent as bounding boxes.
type SpatialExtent struct {
	// One or more bounding boxes that describe the spatial extent of the dataset. In the Core only a single bounding box is supported. Extensions may support additional areas. If multiple areas are provided, the union of the bounding boxes describes the spatial extent.
	BBox [][]float64 `json:"bbox"`
}

// TemporalExtent represents the temporal extent as intervals.
// Each interval is a pair [start, end] where either may be nil (open-ended).
type TemporalExtent struct {
	// One time interval that describe the temporal extent of the dataset.
	Interval [][2]time.Time `json:"interval"`
}

// Collection represents a STAC Collection object.
type Collection struct {
	Type        string                  `json:"type"` // Always "Collection".
	StacVersion string                  `json:"stac_version"`
	ID          string                  `json:"id"`
	Title       *string                 `json:"title,omitempty"`
	Description string                  `json:"description"`
	License     string                  `json:"license,omitempty"`
	ItemType    *string                 `json:"itemType,omitempty"` // Default "Feature".
	Extent      Extent                  `json:"extent"`
	Providers   []Provider              `json:"providers,omitempty"`
	Summaries   map[string]any          `json:"summaries,omitempty"`
	Assets      map[string]FeatureAsset `json:"assets,omitempty"`
	Created     time.Time               `json:"created"`
	Updated     time.Time               `json:"updated"`
	Links       []Link                  `json:"links"`
}

// CollectionsResponse represents the paginated response from GET /collections.
type CollectionsResponse struct {
	Collections []Collection `json:"collections"`
	Links       []Link       `json:"links"`
}

// FeatureProperties holds the core metadata fields of a STAC Feature.
// One of Datetime or the pair StartDatetime/EndDatetime is always set.
type FeatureProperties struct {
	Created       time.Time  `json:"created"`
	Updated       time.Time  `json:"updated"`
	Datetime      *time.Time `json:"datetime"` // Timestamp of when this file is valid, RFC 3339.
	StartDatetime *time.Time `json:"start_datetime,omitempty"`
	EndDatetime   *time.Time `json:"end_datetime,omitempty"`
	Expires       *time.Time `json:"expires,omitempty"`
	Title         *string    `json:"title,omitempty"`
	// Forecast extension fields. See [ForecastFeatureProperties].
	ForecastReferenceDatetime *time.Time `json:"forecast:reference_datetime,omitempty"`
	ForecastHorizon           *string    `json:"forecast:horizon,omitempty"`
	ForecastDuration          *string    `json:"forecast:duration,omitempty"`
	ForecastVariable          *string    `json:"forecast:variable,omitempty"`
	ForecastPerturbed         *bool      `json:"forecast:perturbed,omitempty"`
}

// ForecastFeatureProperties holds the forecast extension fields from
// [FeatureProperties] as non-nullable values. Use [FeatureProperties.Forecast]
// to convert.
type ForecastFeatureProperties struct {
	// Datetime is the timestamp at which the forecast values are valid (RFC 3339).
	Datetime time.Time
	// ReferenceDatetime is the model run initialisation time (RFC 3339).
	ReferenceDatetime time.Time
	// Horizon is the duration between ReferenceDatetime and Datetime as an
	// ISO 8601 duration string (e.g. "PT1H").
	Horizon string
	// Variable is the forecast variable name (e.g. "T_2M").
	Variable string
	// Perturbed is true for perturbed ensemble members, false for control runs.
	Perturbed bool
}

// Forecast extracts the forecast extension fields from p into a
// [ForecastFeatureProperties] with non-nullable values. Nil pointer fields
// are replaced with their zero values.
func (p *FeatureProperties) Forecast() ForecastFeatureProperties {
	fp := ForecastFeatureProperties{}
	if p.Datetime != nil {
		fp.Datetime = *p.Datetime
	}
	if p.ForecastReferenceDatetime != nil {
		fp.ReferenceDatetime = *p.ForecastReferenceDatetime
	}
	if p.ForecastHorizon != nil {
		fp.Horizon = *p.ForecastHorizon
	}
	if p.ForecastVariable != nil {
		fp.Variable = *p.ForecastVariable
	}
	if p.ForecastPerturbed != nil {
		fp.Perturbed = *p.ForecastPerturbed
	}
	return fp
}

// Feature represents a STAC Feature (GeoJSON Feature).
type Feature struct {
	Type           string   `json:"type"`
	StacVersion    string   `json:"stac_version"`
	StacExtensions []string `json:"stac_extensions,omitempty"`
	ID             string   `json:"id"`
	Collection     string   `json:"collection,omitempty"`
	// GeoJSON Point (object) or GeoJSON LineString (object) or GeoJSON Polygon (object) or GeoJSON MultiPoint (object) or GeoJSON MultiLineString (object) or GeoJSON MultiPolygon (object) (itemGeometry)
	Geometry   *geojson.Geometry       `json:"geometry"`
	BBox       []float64               `json:"bbox,omitempty"`
	Properties FeatureProperties       `json:"properties"`
	Assets     map[string]FeatureAsset `json:"assets,omitempty"`
	Links      []Link                  `json:"links"`
}

// ItemsResponse represents the paginated response from GET /collections/{id}/items.
type ItemsResponse struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
	Links    []Link    `json:"links"`
}

// Asset represents a STAC asset object from GET /collections/{id}/assets/{assetId}.
type Asset struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	Title           *string   `json:"title,omitempty"`
	Description     *string   `json:"description,omitempty"`
	Href            *string   `json:"href,omitempty"`
	FileChecksum    *string   `json:"file:checksum,omitempty"`
	Roles           []string  `json:"roles,omitempty"`
	GeoadminVariant *string   `json:"geoadmin:variant,omitempty"`
	GeoadminLang    *string   `json:"geoadmin:lang,omitempty"`
	ProjEPSG        *int      `json:"proj:epsg,omitempty"`
	GSD             *float64  `json:"gsd,omitempty"`
	Created         time.Time `json:"created"`
	Updated         time.Time `json:"updated"`
	Links           []Link    `json:"links,omitempty"`
}

// FeatureAsset represents a STAC asset object from /collections/{id}/items/{featureID}.
type FeatureAsset struct {
	Type            string    `json:"type"`
	Title           *string   `json:"title,omitempty"`
	Description     *string   `json:"description,omitempty"`
	Href            *string   `json:"href,omitempty"`
	FileChecksum    *string   `json:"file:checksum,omitempty"`
	Roles           []string  `json:"roles,omitempty"`
	GeoadminVariant *string   `json:"geoadmin:variant,omitempty"`
	GeoadminLang    *string   `json:"geoadmin:lang,omitempty"`
	ProjEPSG        *int      `json:"proj:epsg,omitempty"`
	GSD             *float64  `json:"gsd,omitempty"`
	Created         time.Time `json:"created"`
	Updated         time.Time `json:"updated"`
	Links           []Link    `json:"links,omitempty"`
}

// AssetsResponse represents the response from GET /collections/{id}/assets.
type AssetsResponse struct {
	Assets []Asset `json:"assets"`
	Links  []Link  `json:"links"`
}

// Catalog represents the core STAC Catalog object.
type Catalog struct {
	Type        string   `json:"type"`
	StacVersion string   `json:"stac_version"`
	ID          string   `json:"id"`
	Title       *string  `json:"title,omitempty"`
	Description string   `json:"description"`
	ConformsTo  []string `json:"conformsTo,omitempty"`
	Links       []Link   `json:"links"`
}
