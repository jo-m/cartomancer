package forecast

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErythemalUVFromSW(t *testing.T) {
	tests := []struct {
		name          string
		directSW      float64
		diffuseSW     float64
		wantNaN       bool
		wantErythemal float64
		wantUVIApprox float64
		uviTolerance  float64
	}{
		{
			name:          "clear-sky noon, ~1000 W/m^2 -> UVI ~10",
			directSW:      800,
			diffuseSW:     200,
			wantErythemal: 0.25,
			wantUVIApprox: 10,
			uviTolerance:  0.01,
		},
		{
			name:          "overcast, mostly diffuse",
			directSW:      40,
			diffuseSW:     160,
			wantErythemal: 0.05,
			wantUVIApprox: 2,
			uviTolerance:  0.01,
		},
		{
			name:          "night, zero",
			directSW:      0,
			diffuseSW:     0,
			wantErythemal: 0,
			wantUVIApprox: 0,
			uviTolerance:  0.0001,
		},
		{
			name:     "NaN direct",
			directSW: math.NaN(),
			wantNaN:  true,
		},
		{
			name:      "NaN diffuse",
			diffuseSW: math.NaN(),
			wantNaN:   true,
		},
		{
			name:      "negative total irradiance is rejected",
			directSW:  -10,
			diffuseSW: 0,
			wantNaN:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ery := ErythemalUVFromSW(tt.directSW, tt.diffuseSW)
			if tt.wantNaN {
				require.True(t, math.IsNaN(ery))
				return
			}
			require.InDelta(t, tt.wantErythemal, ery, 1e-6)
			uvi := UVIndexFromErythemalUV(ery)
			require.InDelta(t, tt.wantUVIApprox, uvi, tt.uviTolerance)
		})
	}
}

func TestUVIndexFromErythemalUVNaN(t *testing.T) {
	require.True(t, math.IsNaN(UVIndexFromErythemalUV(math.NaN())))
}

func TestUVDoseSEDStep(t *testing.T) {
	// At noon, ery UV = 0.25 W/m^2. One hour at that irradiance:
	// dose = 0.25 * 3600 = 900 J/m^2 = 9 SED.
	got := UVDoseSEDStep(0.25, 3600)
	require.InDelta(t, 9.0, got, 1e-9)

	// Half an hour at 0.1 W/m^2 -> 0.1 * 1800 / 100 = 1.8 SED.
	got = UVDoseSEDStep(0.1, 1800)
	require.InDelta(t, 1.8, got, 1e-9)

	// Edge cases return 0 so the value is safe to sum.
	require.Equal(t, 0.0, UVDoseSEDStep(math.NaN(), 3600))
	require.Equal(t, 0.0, UVDoseSEDStep(0.25, 0))
	require.Equal(t, 0.0, UVDoseSEDStep(0.25, -10))
	require.Equal(t, 0.0, UVDoseSEDStep(-0.1, 3600))
}
