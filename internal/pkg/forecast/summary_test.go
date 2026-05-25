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

func TestComputeSummary_EmptyHandle(t *testing.T) {
	h := &Handle{
		values: map[string][]timedValues{},
	}
	pts := track.Points{
		{Lat: 46.0, Lon: 8.0, Distance: 0},
		{Lat: 46.1, Lon: 8.1, Distance: 13000},
	}
	bearings := pts.Bearings()
	s := computeSummary(h, pts, bearings, fixedTime(), 7.78)

	require.True(t, math.IsNaN(s.avgTempC), "expected NaN for temperature with no data")
	require.True(t, math.IsNaN(s.windHeadMs), "expected NaN for wind with no data")
	require.Equal(t, 0.0, s.totalPrecipMm)
	require.Equal(t, 0.0, s.uvDoseSED, "expected zero UV dose with no data")
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
