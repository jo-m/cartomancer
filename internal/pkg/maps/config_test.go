package maps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapsConfig_Enabled(t *testing.T) {
	require.False(t, (&MapsConfig{}).Enabled())
	require.True(t, (&MapsConfig{MapsBbox: "5.5,45.5,11.0,48.2"}).Enabled())
}

func TestMapsConfig_Validate_disabled(t *testing.T) {
	require.NoError(t, (&MapsConfig{}).Validate())
}

func TestMapsConfig_Validate_ok(t *testing.T) {
	c := MapsConfig{MapsBbox: testBbox, MapsMaxZoom: DefaultMaxZoom}
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

func TestMapsConfig_Validate_emptyBbox(t *testing.T) {
	c := MapsConfig{MapsBbox: "", MapsMaxZoom: 8}
	// Empty bbox means disabled, so validation passes.
	require.NoError(t, c.Validate())
}
