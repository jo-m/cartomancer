package ktzh

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/paulmach/orb/geojson"

	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
)

// Feature is one decoded street light from the canton's OGD layer.
type Feature struct {
	// SourceID is a stable identifier prefixed with "ktzh-" to avoid
	// collisions with other providers. The upstream GeoJSON has no feature
	// id, so it is derived from the geodb_oid attribute.
	SourceID string

	// Properties holds the source feature's attributes verbatim as a JSON
	// object.
	Properties json.RawMessage

	// Geometry is the lamp location in WGS84 (GeoJSON axis order: lon, lat).
	// It is nil for source features without a geometry; callers should skip
	// such features.
	Geometry *geojson.Geometry
}

// featureProps mirrors the parts of the ogd-0124 schema needed to derive the
// source id. All other attributes are stored verbatim.
type featureProps struct {
	GeodbOID int `json:"geodb_oid"`
}

// decodeFeature converts one raw GeoJSON feature into a [Feature].
// It fails when the feature has no usable geodb_oid, because that attribute
// is the only stable identity the source offers.
func decodeFeature(raw wfs.GeoJSONFeature) (Feature, error) {
	var props featureProps
	if len(raw.Properties) > 0 {
		if err := json.Unmarshal(raw.Properties, &props); err != nil {
			return Feature{}, fmt.Errorf("decode properties: %w", err)
		}
	}
	if props.GeodbOID == 0 {
		return Feature{}, errors.New("missing geodb_oid")
	}

	return Feature{
		SourceID:   fmt.Sprintf("ktzh-%d", props.GeodbOID),
		Properties: raw.Properties,
		Geometry:   raw.Geometry,
	}, nil
}
