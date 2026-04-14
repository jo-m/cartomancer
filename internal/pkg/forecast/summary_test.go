package forecast

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

func TestRelativeWindSector(t *testing.T) {
	tests := []struct {
		name    string
		relDeg  float64
		wantSec int
	}{
		{"headwind at 0", 0, 0},
		{"headwind at 350", 350, 0},
		{"headwind at 20", 20, 0},
		{"right at 90", 90, 1},
		{"right at 50", 50, 1},
		{"right at 130", 130, 1},
		{"tailwind at 180", 180, 2},
		{"tailwind at 140", 140, 2},
		{"tailwind at 220", 220, 2},
		{"left at 270", 270, 3},
		{"left at 230", 230, 3},
		{"left at 310", 310, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeWindSector(tt.relDeg)
			require.Equal(t, tt.wantSec, got, "relDeg=%f", tt.relDeg)
		})
	}
}

func TestForwardBearing(t *testing.T) {
	// Due east: Bern to Zurich (roughly east).
	b := forwardBearing(46.948, 7.447, 47.376, 8.541)
	require.Greater(t, b, 30.0)
	require.Less(t, b, 80.0)

	// Due north.
	b = forwardBearing(46.0, 8.0, 47.0, 8.0)
	require.InDelta(t, 0, b, 1.0)

	// Due south.
	b = forwardBearing(47.0, 8.0, 46.0, 8.0)
	require.InDelta(t, 180, b, 1.0)
}

func TestComputeDistancesAndBearings(t *testing.T) {
	pts := track.Points{
		{Lat: 46.0, Lon: 8.0},
		{Lat: 46.0, Lon: 8.01},
		{Lat: 46.0, Lon: 8.02},
	}
	distances, bearings := computeDistancesAndBearings(pts)

	require.Len(t, distances, 3)
	require.Len(t, bearings, 3)

	require.Equal(t, 0.0, distances[0])
	require.Greater(t, distances[1], 500.0)
	require.Greater(t, distances[2], distances[1])

	// Heading east, bearings should be around 90 degrees.
	for _, b := range bearings {
		require.InDelta(t, 90, b, 5.0)
	}
}

func TestComputeSummary_EmptyHandle(t *testing.T) {
	h := &Handle{
		values: map[string][]timedValues{},
	}
	pts := track.Points{
		{Lat: 46.0, Lon: 8.0},
		{Lat: 46.1, Lon: 8.1},
	}
	distances, bearings := computeDistancesAndBearings(pts)
	s := computeSummary(h, pts, distances, bearings, fixedTime(), 7.78)

	require.True(t, math.IsNaN(s.avgTempC), "expected NaN for temperature with no data")
	require.True(t, math.IsNaN(s.windHeadMs), "expected NaN for wind with no data")
	require.Equal(t, 0.0, s.totalPrecipMm)
}

func TestNullFloat(t *testing.T) {
	nf := nullFloat(math.NaN())
	require.False(t, nf.Valid)

	nf = nullFloat(12.5)
	require.True(t, nf.Valid)
	require.Equal(t, 12.5, nf.Float64)
}

func TestNextFullHour(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want time.Time
	}{
		{
			"mid-hour rounds up",
			time.Date(2026, 3, 22, 9, 34, 12, 0, time.UTC),
			time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
		},
		{
			"exact hour stays",
			time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
		},
		{
			"one second past rounds up",
			time.Date(2026, 3, 22, 10, 0, 1, 0, time.UTC),
			time.Date(2026, 3, 22, 11, 0, 0, 0, time.UTC),
		},
		{
			"23:30 rolls to midnight",
			time.Date(2026, 3, 22, 23, 30, 0, 0, time.UTC),
			time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextFullHour(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func fixedTime() time.Time {
	return time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
}
