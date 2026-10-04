package stadtzh

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/paulmach/orb/geojson"

	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
)

// Feature is one decoded street light from the city's OGD layer.
type Feature struct {
	// SourceID is a stable identifier prefixed with "stadtzh-" to avoid
	// collisions with other providers. It is the upstream feature id, falling
	// back to the objectid attribute.
	SourceID string

	// Properties holds the source feature's attributes verbatim as a JSON
	// object.
	Properties json.RawMessage

	// Geometry is the lamp location in WGS84 (GeoJSON axis order: lon, lat).
	// It is nil for source features without a geometry; callers should skip
	// such features.
	Geometry *geojson.Geometry
}

// featureProps mirrors the parts of the ewz_brennstelle_p schema needed to
// derive the source id when the feature id is missing. All other attributes
// are stored verbatim.
type featureProps struct {
	ObjectID int `json:"objectid"`
}

// decodeFeature converts one raw GeoJSON feature into a [Feature].
// It falls back to the objectid attribute when the feature has no id, and
// fails when neither is usable.
func decodeFeature(raw wfs.GeoJSONFeature) (Feature, error) {
	sourceID := raw.ID
	if sourceID == "" {
		var props featureProps
		if len(raw.Properties) > 0 {
			if err := json.Unmarshal(raw.Properties, &props); err != nil {
				return Feature{}, fmt.Errorf("decode properties: %w", err)
			}
		}
		if props.ObjectID == 0 {
			return Feature{}, errors.New("feature has neither an id nor an objectid")
		}
		sourceID = strconv.Itoa(props.ObjectID)
	}

	return Feature{
		SourceID:   "stadtzh-" + sourceID,
		Properties: raw.Properties,
		Geometry:   raw.Geometry,
	}, nil
}
