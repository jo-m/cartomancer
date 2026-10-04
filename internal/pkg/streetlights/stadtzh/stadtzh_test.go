package stadtzh

import (
	"encoding/json"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
)

// sampleProps is a verbatim property set from the live endpoint.
const sampleProps = `{"art":0,"art_txt":"Beleuchtung","geometrie_gdo":null,"nisnr":"LEU119686","objectid":101,"objid":"101","orientierung":59.2}`

// sampleFeature builds the raw GeoJSON feature shape produced by
// [Fetch] for the sample properties.
func sampleFeature(t *testing.T) wfs.GeoJSONFeature {
	t.Helper()
	geom, err := geojson.UnmarshalGeometry([]byte(`{"type":"Point","coordinates":[8.546895,47.42932]}`))
	require.NoError(t, err)
	return wfs.GeoJSONFeature{
		ID:         "ewz_brennstelle_p.101",
		Properties: json.RawMessage(sampleProps),
		Geometry:   geom,
	}
}

func TestDecodeFeature(t *testing.T) {
	f, err := decodeFeature(sampleFeature(t))
	require.NoError(t, err)
	require.Equal(t, "stadtzh-ewz_brennstelle_p.101", f.SourceID)

	p, ok := f.Geometry.Geometry().(orb.Point)
	require.True(t, ok)
	require.InDelta(t, 8.546895, p.Lon(), 1e-9)
	require.InDelta(t, 47.42932, p.Lat(), 1e-9)
}

func TestDecodeFeatureWithoutIDFallsBackToObjectID(t *testing.T) {
	raw := sampleFeature(t)
	raw.ID = ""
	f, err := decodeFeature(raw)
	require.NoError(t, err)
	require.Equal(t, "stadtzh-101", f.SourceID)
}

func TestDecodeFeatureWithoutIDAndObjectID(t *testing.T) {
	raw := sampleFeature(t)
	raw.ID = ""
	raw.Properties = json.RawMessage(`{"nisnr":"LEU119686"}`)
	_, err := decodeFeature(raw)
	require.ErrorContains(t, err, "neither an id nor an objectid")
}

func TestDecodeFeatureInvalidProperties(t *testing.T) {
	raw := sampleFeature(t)
	raw.ID = ""
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
