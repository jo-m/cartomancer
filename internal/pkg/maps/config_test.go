package maps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapsConfig_Validate_defaults(t *testing.T) {
	require.NoError(t, (&MapsConfig{MapsMaxZoom: testMaxZoom}).Validate())
}

func TestMapsConfig_Validate_withBbox(t *testing.T) {
	c := MapsConfig{MapsBbox: testBbox, MapsMaxZoom: testMaxZoom}
	require.NoError(t, c.Validate())
}

func TestMapsConfig_Validate_emptyBbox(t *testing.T) {
	c := MapsConfig{MapsBbox: "", MapsMaxZoom: 8}
	require.NoError(t, c.Validate())
}

func TestMapsConfig_Validate_badZoom(t *testing.T) {
	c := MapsConfig{MapsBbox: testBbox, MapsMaxZoom: 23}
	require.Error(t, c.Validate())
}

func TestMapsConfig_Validate_maxZoom22(t *testing.T) {
	c := MapsConfig{MapsBbox: testBbox, MapsMaxZoom: 22}
	require.NoError(t, c.Validate())
}

func TestMapsConfig_Validate_badBbox(t *testing.T) {
	c := MapsConfig{MapsBbox: "1,2,3", MapsMaxZoom: 8}
	require.Error(t, c.Validate())
}

func TestMapsConfig_ParsedBbox_empty(t *testing.T) {
	c := MapsConfig{MapsBbox: ""}
	b, err := c.ParsedBbox()
	require.NoError(t, err)
	require.Nil(t, b)
}

func TestMapsConfig_ParsedBbox_set(t *testing.T) {
	c := MapsConfig{MapsBbox: testBbox}
	b, err := c.ParsedBbox()
	require.NoError(t, err)
	require.NotNil(t, b)
	require.InDelta(t, 5.5, b.MinLon, 0.001)
	require.InDelta(t, 45.5, b.MinLat, 0.001)
	require.InDelta(t, 11.0, b.MaxLon, 0.001)
	require.InDelta(t, 48.2, b.MaxLat, 0.001)
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
