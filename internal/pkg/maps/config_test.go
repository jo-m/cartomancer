package maps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapsConfig_Enabled(t *testing.T) {
	require.False(t, (&MapsConfig{}).Enabled())
	require.True(t, (&MapsConfig{MapsDir: "/tmp/maps"}).Enabled())
}

func TestMapsConfig_Validate_disabled(t *testing.T) {
	require.NoError(t, (&MapsConfig{}).Validate())
}

func TestMapsConfig_Validate_ok(t *testing.T) {
	c := MapsConfig{MapsDir: "/tmp/maps", MapsBbox: DefaultBbox, MapsMaxZoom: DefaultMaxZoom}
	require.NoError(t, c.Validate())
}

func TestMapsConfig_Validate_badZoom(t *testing.T) {
	c := MapsConfig{MapsDir: "/tmp/maps", MapsBbox: DefaultBbox, MapsMaxZoom: 20}
	require.Error(t, c.Validate())
}

func TestMapsConfig_Validate_emptyBbox(t *testing.T) {
	c := MapsConfig{MapsDir: "/tmp/maps", MapsBbox: "", MapsMaxZoom: 8}
	require.Error(t, c.Validate())
}
