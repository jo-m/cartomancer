package lv95

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// degreeDelta is the tolerance used for reference comparisons. ~1e-4 degrees
// corresponds to roughly 10 m on the ground at Swiss latitudes, which sits
// comfortably above the approximation error of the swisstopo formulas
// (better than 0.12" in longitude and 0.08" in latitude).
const degreeDelta = 1e-4

// TestToWGS84ReferencePoints checks the conversion against truth values
// obtained from swisstopo's REFRAME web service
// (https://geodesy.geo.admin.ch/reframe/lv95towgs84).
func TestToWGS84ReferencePoints(t *testing.T) {
	tests := []struct {
		name             string
		easting          float64
		northing         float64
		wantLon, wantLat float64
	}{
		{
			name:     "bern federal palace",
			easting:  2600000,
			northing: 1200000,
			wantLon:  7.438632495274896,
			wantLat:  46.951082876677035,
		},
		{
			name:     "zurich hb",
			easting:  2683110,
			northing: 1247843,
			wantLon:  8.539117632027276,
			wantLat:  47.376181500308356,
		},
		{
			name:     "st gallen",
			easting:  2745945,
			northing: 1254160,
			wantLon:  9.372937990893979,
			wantLat:  47.42205784885163,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lon, lat := ToWGS84(tc.easting, tc.northing)
			require.InDelta(t, tc.wantLon, lon, degreeDelta)
			require.InDelta(t, tc.wantLat, lat, degreeDelta)
		})
	}
}

// TestToWGS84SwisstopoExample reproduces the worked numerical example from
// section 2 of the swisstopo PDF "Approximate formulas for the transformation
// between Swiss projection coordinates and WGS84" (December 2016):
// E = 2 700 000 m, N = 1 100 000 m -> lambda = 8 deg 43' 49.80",
// phi = 46 deg 02' 38.86".
func TestToWGS84SwisstopoExample(t *testing.T) {
	const (
		// 8 deg + 43/60 + 49.80/3600 = 8.7305000.
		wantLon = 8.7305
		// 46 deg + 2/60 + 38.86/3600 = 46.04412777...
		wantLat = 46.044127777777778
	)
	lon, lat := ToWGS84(2700000, 1100000)
	require.InDelta(t, wantLon, lon, degreeDelta)
	require.InDelta(t, wantLat, lat, degreeDelta)
}
