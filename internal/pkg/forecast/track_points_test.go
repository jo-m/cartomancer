package forecast

import (
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

func TestInterpolatedTrackPoints_InvalidStep(t *testing.T) {
	encoded, err := track.EncodeVarint(track.Points{
		{Lat: 46.0, Lon: 7.0},
		{Lat: 46.1, Lon: 7.1},
	})
	require.NoError(t, err)

	tr := db.Track{PolylineDp50mVarint: encoded}
	_, err = InterpolatedTrackPoints(tr, 0)
	require.Error(t, err)
}

func TestInterpolatedTrackPoints_FixedSpacing(t *testing.T) {
	// Three points along the equator at known cumulative distances.
	pts := track.Points{
		{Lat: 0, Lon: 0, Distance: 0},
		{Lat: 0, Lon: 1, Distance: 111_000},
		{Lat: 0, Lon: 2, Distance: 222_000},
	}
	encoded, err := track.EncodeVarint(pts)
	require.NoError(t, err)

	tr := db.Track{PolylineDp50mVarint: encoded}
	got, err := InterpolatedTrackPoints(tr, 50_000)
	require.NoError(t, err)
	require.Greater(t, len(got), 4)

	// First point matches the source.
	require.InDelta(t, 0.0, got[0].Distance, 1e-6)
	require.InDelta(t, 0.0, got[0].Lat, 1e-9)
	require.InDelta(t, 0.0, got[0].Lon, 1e-9)

	// Distances are monotonically increasing.
	for i := 1; i < len(got); i++ {
		require.Greater(t, got[i].Distance, got[i-1].Distance)
	}

	// Intermediate gaps are exactly 50 km.
	for i := 1; i < len(got)-1; i++ {
		gap := got[i].Distance - got[i-1].Distance
		require.InDelta(t, 50_000, gap, 1e-6)
	}

	// Final point matches the source's last point.
	last := got[len(got)-1]
	require.InDelta(t, 0.0, last.Lat, 1e-9)
	require.InDelta(t, 2.0, last.Lon, 1e-9)
	require.InDelta(t, 222_000, last.Distance, 1e-6)
}

func TestInterpolatedTrackPoints_TooShort(t *testing.T) {
	encoded, err := track.EncodeVarint(track.Points{{Lat: 46.0, Lon: 7.0}})
	require.NoError(t, err)

	tr := db.Track{PolylineDp50mVarint: encoded}
	got, err := InterpolatedTrackPoints(tr, LiveStepM)
	require.NoError(t, err)
	require.Nil(t, got)
}
