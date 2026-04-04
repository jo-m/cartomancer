package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeSunEvents(t *testing.T) {
	// Zurich: ~47.37N, 8.55E. In summer, sunrise ~05:30, sunset ~21:00 UTC.
	lat, lon := 47.37, 8.55
	totalDist := 100000.0 // 100 km

	t.Run("ride spanning sunrise", func(t *testing.T) {
		// Ride from 02:00 to 09:00 UTC on June 21 (dawn ~03:15, sunrise ~05:30 in Zurich).
		start := time.Date(2025, 6, 21, 2, 0, 0, 0, time.UTC)
		end := time.Date(2025, 6, 21, 9, 0, 0, 0, time.UTC)

		events := computeSunEvents(start, end, lat, lon, totalDist)
		require.NotEmpty(t, events)

		types := make(map[string]bool)
		for _, e := range events {
			types[e.Type] = true
			assert.True(t, e.DistanceM >= 0 && e.DistanceM <= totalDist,
				"distance %f should be within [0, %f]", e.DistanceM, totalDist)

			et, err := time.Parse(time.RFC3339, e.Time)
			require.NoError(t, err)
			assert.True(t, !et.Before(start) && !et.After(end),
				"event time %s should be within ride window", e.Time)
		}
		assert.True(t, types["sunrise"], "should contain sunrise event")
		assert.True(t, types["dawn"], "should contain dawn event")
	})

	t.Run("ride during midday has no events", func(t *testing.T) {
		// Ride from 10:00 to 14:00 UTC on June 21 - no sunrise/sunset.
		start := time.Date(2025, 6, 21, 10, 0, 0, 0, time.UTC)
		end := time.Date(2025, 6, 21, 14, 0, 0, 0, time.UTC)

		events := computeSunEvents(start, end, lat, lon, totalDist)
		assert.Empty(t, events)
	})

	t.Run("events are sorted by distance", func(t *testing.T) {
		// Long ride spanning both sunrise and sunset.
		start := time.Date(2025, 6, 21, 3, 0, 0, 0, time.UTC)
		end := time.Date(2025, 6, 21, 22, 0, 0, 0, time.UTC)

		events := computeSunEvents(start, end, lat, lon, totalDist)
		require.True(t, len(events) >= 2, "expected multiple sun events")

		for i := 1; i < len(events); i++ {
			assert.LessOrEqual(t, events[i-1].DistanceM, events[i].DistanceM,
				"events should be sorted by distance")
		}
	})
}
