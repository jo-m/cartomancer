package api

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForwardBearing(t *testing.T) {
	tests := []struct {
		name         string
		lat1, lon1   float64
		lat2, lon2   float64
		wantDeg      float64
		toleranceDeg float64
	}{
		{"due north", 47.0, 8.0, 48.0, 8.0, 0, 0.5},
		{"due east", 47.0, 8.0, 47.0, 9.0, 90, 1},
		{"due south", 48.0, 8.0, 47.0, 8.0, 180, 0.5},
		{"due west", 47.0, 9.0, 47.0, 8.0, 270, 1},
		{"northeast", 47.0, 8.0, 48.0, 9.0, 35, 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := forwardBearing(tc.lat1, tc.lon1, tc.lat2, tc.lon2)
			diff := math.Abs(got - tc.wantDeg)
			if diff > 180 {
				diff = 360 - diff
			}
			assert.LessOrEqual(t, diff, tc.toleranceDeg, "bearing: got %.1f, want ~%.1f", got, tc.wantDeg)
		})
	}
}
