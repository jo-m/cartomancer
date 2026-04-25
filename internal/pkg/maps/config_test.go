package maps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapsConfig_Validate_defaults(t *testing.T) {
	require.NoError(t, (&MapsConfig{MapsSpecs: "@7;-16.9,27.8,-13.4,29.2@10"}).Validate())
}

func TestMapsConfig_Validate_empty(t *testing.T) {
	require.NoError(t, (&MapsConfig{MapsSpecs: ""}).Validate())
}

func TestMapsConfig_Validate_worldOnly(t *testing.T) {
	require.NoError(t, (&MapsConfig{MapsSpecs: "@7"}).Validate())
}

func TestMapsConfig_Validate_multiple(t *testing.T) {
	c := MapsConfig{MapsSpecs: "@7;5.5,45.5,11.0,48.2@10;6.0,46.0,10.0,47.5@14"}
	require.NoError(t, c.Validate())
}

func TestMapsConfig_Validate_badZoom(t *testing.T) {
	c := MapsConfig{MapsSpecs: "5.5,45.5,11.0,48.2@23"}
	require.Error(t, c.Validate())
}

func TestMapsConfig_Validate_maxZoom22(t *testing.T) {
	c := MapsConfig{MapsSpecs: "5.5,45.5,11.0,48.2@22"}
	require.NoError(t, c.Validate())
}

func TestMapsConfig_Validate_badBbox(t *testing.T) {
	c := MapsConfig{MapsSpecs: "1,2,3@8"}
	require.Error(t, c.Validate())
}

func TestMapsConfig_Validate_missingSeparator(t *testing.T) {
	c := MapsConfig{MapsSpecs: "5.5,45.5,11.0,48.2"}
	require.Error(t, c.Validate())
}

func TestMapsConfig_ParsedSpecs_empty(t *testing.T) {
	c := MapsConfig{MapsSpecs: ""}
	specs, err := c.ParsedSpecs()
	require.NoError(t, err)
	require.Nil(t, specs)
}

func TestMapsConfig_ParsedSpecs_worldOnly(t *testing.T) {
	c := MapsConfig{MapsSpecs: "@7"}
	specs, err := c.ParsedSpecs()
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Nil(t, specs[0].Bbox)
	require.Equal(t, 7, specs[0].MaxZoom)
}

func TestMapsConfig_ParsedSpecs_multiple(t *testing.T) {
	c := MapsConfig{MapsSpecs: "@7;5.5,45.5,11.0,48.2@10"}
	specs, err := c.ParsedSpecs()
	require.NoError(t, err)
	require.Len(t, specs, 2)

	require.Nil(t, specs[0].Bbox)
	require.Equal(t, 7, specs[0].MaxZoom)

	require.NotNil(t, specs[1].Bbox)
	require.InDelta(t, 5.5, specs[1].Bbox.MinLon, 0.001)
	require.InDelta(t, 45.5, specs[1].Bbox.MinLat, 0.001)
	require.InDelta(t, 11.0, specs[1].Bbox.MaxLon, 0.001)
	require.InDelta(t, 48.2, specs[1].Bbox.MaxLat, 0.001)
	require.Equal(t, 10, specs[1].MaxZoom)
}

func TestParseMapSpec_world(t *testing.T) {
	spec, err := parseMapSpec("@7")
	require.NoError(t, err)
	require.Nil(t, spec.Bbox)
	require.Equal(t, 7, spec.MaxZoom)
}

func TestParseMapSpec_regional(t *testing.T) {
	spec, err := parseMapSpec("5.5,45.5,11.0,48.2@10")
	require.NoError(t, err)
	require.NotNil(t, spec.Bbox)
	require.Equal(t, Bbox{MinLon: 5.5, MinLat: 45.5, MaxLon: 11.0, MaxLat: 48.2}, *spec.Bbox)
	require.Equal(t, 10, spec.MaxZoom)
}

func TestParseMapSpec_invalidZoom(t *testing.T) {
	_, err := parseMapSpec("5.5,45.5,11.0,48.2@abc")
	require.Error(t, err)
}

func TestParseMapSpec_noSeparator(t *testing.T) {
	_, err := parseMapSpec("5.5,45.5,11.0,48.2")
	require.Error(t, err)
}

func TestParseBbox(t *testing.T) {
	b, err := ParseBbox("5.5,45.5,11.0,48.2")
	require.NoError(t, err)
	require.Equal(t, Bbox{MinLon: 5.5, MinLat: 45.5, MaxLon: 11.0, MaxLat: 48.2}, b)
}

func TestParseBbox_invalid(t *testing.T) {
	_, err := ParseBbox("not,a,bbox,x")
	require.Error(t, err)
}

func TestBbox_String(t *testing.T) {
	b := Bbox{MinLon: 5.5, MinLat: 45.5, MaxLon: 11, MaxLat: 48.2}
	require.Equal(t, "5.5,45.5,11,48.2", b.String())
}
