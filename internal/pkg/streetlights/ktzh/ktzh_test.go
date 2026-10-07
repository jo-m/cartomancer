package ktzh

import (
	"encoding/json"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
)

// sampleProps is a verbatim property set from the live endpoint.
const sampleProps = `{"nummer":"5141","ausgeschaltet_txt":"Beleuchtung eingeschaltet","beleuchtungstyp_txt":"EKZ Import","bauart_txt":"Stahl mit Ausleger","geodb_oid":101}`

// sampleFeature builds the raw GeoJSON feature shape produced by
// [Fetch] for the sample properties.
func sampleFeature(t *testing.T) wfs.GeoJSONFeature {
	t.Helper()
	geom, err := geojson.UnmarshalGeometry([]byte(`{"type":"Point","coordinates":[8.594926637640667,47.346301755015453]}`))
	require.NoError(t, err)
	return wfs.GeoJSONFeature{
		Properties: json.RawMessage(sampleProps),
		Geometry:   geom,
	}
}

func TestDecodeFeature(t *testing.T) {
	f, err := decodeFeature(sampleFeature(t))
	require.NoError(t, err)
	require.Equal(t, "ktzh-101", f.SourceID)

	p, ok := f.Geometry.Geometry().(orb.Point)
	require.True(t, ok)
	require.InDelta(t, 8.594926637640667, p.Lon(), 1e-9)
	require.InDelta(t, 47.346301755015453, p.Lat(), 1e-9)
}

func TestDecodeFeatureWithoutGeodbOID(t *testing.T) {
	raw := sampleFeature(t)
	raw.Properties = json.RawMessage(`{"nummer":"5141"}`)
	_, err := decodeFeature(raw)
	require.ErrorContains(t, err, "missing geodb_oid")
}

func TestDecodeFeatureInvalidProperties(t *testing.T) {
	raw := sampleFeature(t)
	raw.Properties = json.RawMessage(`not json`)
	_, err := decodeFeature(raw)
	require.ErrorContains(t, err, "decode properties")
}

func TestDecodeFeatureWithoutGeometry(t *testing.T) {
	raw := sampleFeature(t)
	raw.Geometry = nil
	f, err := decodeFeature(raw)
	require.NoError(t, err)
	require.Nil(t, f.Geometry, "features without geometry are skipped later, not rejected")
}
