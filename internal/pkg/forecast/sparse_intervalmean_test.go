package forecast

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSampleIntervalMean verifies de-averaging of variables stored as
// cumulative means from the forecast reference time, e.g. ASWDIR_S.
func TestSampleIntervalMean(t *testing.T) {
	ref := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	h := &Handle{
		ReferenceTime: ref,
		values: map[string][]timedValues{
			// Cumulative-mean SW at one grid point over 3 successive hours.
			// c(1h) = 100  -> integral over [0,1h] = 100 * 1 = 100 (interval mean 100).
			// c(2h) = 250  -> integral over [0,2h] = 500; over [1h,2h] = 400 (interval mean 400).
			// c(3h) = 300  -> integral over [0,3h] = 900; over [2h,3h] = 400 (interval mean 400).
			"ASWDIR_S": {
				{
					validTime:      ref.Add(1 * time.Hour),
					validUntilTime: ref.Add(2 * time.Hour),
					vals:           []float32{100},
				},
				{
					validTime:      ref.Add(2 * time.Hour),
					validUntilTime: ref.Add(3 * time.Hour),
					vals:           []float32{250},
				},
				{
					validTime:      ref.Add(3 * time.Hour),
					validUntilTime: ref.Add(4 * time.Hour),
					vals:           []float32{300},
				},
			},
		},
	}

	got := h.SampleIntervalMean("ASWDIR_S", ref.Add(90*time.Minute), 0)
	require.InDelta(t, 100.0, got, 1e-3, "first entry de-averages against zero")

	got = h.SampleIntervalMean("ASWDIR_S", ref.Add(150*time.Minute), 0)
	require.InDelta(t, 400.0, got, 1e-3, "second entry: 2*250 - 1*100 = 400")

	got = h.SampleIntervalMean("ASWDIR_S", ref.Add(210*time.Minute), 0)
	require.InDelta(t, 400.0, got, 1e-3, "third entry: 3*300 - 2*250 = 400")
}

func TestSampleIntervalMean_OutOfRange(t *testing.T) {
	ref := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	h := &Handle{
		ReferenceTime: ref,
		values: map[string][]timedValues{
			"ASWDIR_S": {
				{
					validTime:      ref.Add(1 * time.Hour),
					validUntilTime: ref.Add(2 * time.Hour),
					vals:           []float32{200},
				},
			},
		},
	}

	require.True(t, math.IsNaN(float64(h.SampleIntervalMean("ASWDIR_S", ref, 0))),
		"time before first entry returns NaN")
	require.True(t, math.IsNaN(float64(h.SampleIntervalMean("ASWDIR_S", ref.Add(3*time.Hour), 0))),
		"time past validUntilTime returns NaN")
	require.True(t, math.IsNaN(float64(h.SampleIntervalMean("UNKNOWN", ref.Add(90*time.Minute), 0))),
		"unknown variable returns NaN")
	require.True(t, math.IsNaN(float64(h.SampleIntervalMean("ASWDIR_S", ref.Add(90*time.Minute), 99))),
		"out-of-range location returns NaN")
}
