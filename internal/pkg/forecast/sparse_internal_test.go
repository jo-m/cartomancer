package forecast

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo/vars"
)

// makeEntry builds a [timedValues] for de-averaging tests. The validUntilTime
// is fixed to validTime+1h, mirroring the convention used by meteo/job.go
// when files are written to the database.
func makeEntry(refTime, intervalStart, validTime time.Time, vals ...float32) timedValues {
	return timedValues{
		validTime:      validTime,
		validUntilTime: validTime.Add(time.Hour),
		referenceTime:  refTime,
		intervalStart:  intervalStart,
		vals:           append([]float32(nil), vals...),
	}
}

// TestDeaverageRunningMean_BasicTwoStep checks that two stacked running-mean
// entries produce the algebraically correct per-step values.
func TestDeaverageRunningMean_BasicTwoStep(t *testing.T) {
	refTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// prev: running mean over [ref, ref+1h] = 100. So total = 100*1 = 100.
	// curr: running mean over [ref, ref+2h] = 120. So total = 120*2 = 240.
	// Step 2 mean = (240 - 100) / 1 = 140.
	entries := []timedValues{
		makeEntry(refTime, refTime, refTime.Add(time.Hour), 100),
		makeEntry(refTime, refTime, refTime.Add(2*time.Hour), 120),
	}

	values := map[string][]timedValues{vars.VarAswdirS.Name: entries}
	deaverageRunningAggregates(values)

	require.InDelta(t, 100.0, float64(values[vars.VarAswdirS.Name][0].vals[0]), 1e-3,
		"first step is one hour wide, value stays unchanged")
	require.InDelta(t, 140.0, float64(values[vars.VarAswdirS.Name][1].vals[0]), 1e-3,
		"second step is de-averaged to (2*120 - 1*100) / 1 = 140")
}

// TestDeaverageRunningMean_MissingPredecessor verifies that an entry whose
// running window is more than one hour wide but has no usable predecessor
// becomes NaN.
func TestDeaverageRunningMean_MissingPredecessor(t *testing.T) {
	refTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	entries := []timedValues{
		// Two-hour-wide running mean, no predecessor: cannot recover step.
		makeEntry(refTime, refTime, refTime.Add(2*time.Hour), 120),
	}

	values := map[string][]timedValues{vars.VarAswdirS.Name: entries}
	deaverageRunningAggregates(values)

	require.True(t, math.IsNaN(float64(values[vars.VarAswdirS.Name][0].vals[0])),
		"missing same-run predecessor must produce NaN")
}

// TestDeaverageRunningMean_DifferentReferenceTime verifies that a predecessor
// from a different model run is not used; the current entry must be NaN.
func TestDeaverageRunningMean_DifferentReferenceTime(t *testing.T) {
	refOld := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	refNew := refOld.Add(6 * time.Hour)

	// Predecessor belongs to refOld; curr belongs to refNew. They must not be
	// combined even though their validTimes line up.
	entries := []timedValues{
		makeEntry(refOld, refOld, refOld.Add(7*time.Hour), 100),
		makeEntry(refNew, refNew, refNew.Add(2*time.Hour), 120),
	}

	values := map[string][]timedValues{vars.VarAswdirS.Name: entries}
	deaverageRunningAggregates(values)

	require.True(t, math.IsNaN(float64(values[vars.VarAswdirS.Name][1].vals[0])),
		"predecessor from a different run must not be used; expect NaN")
}

// TestDeaverageRunningMean_InstantUntouched verifies that variables with
// "Instant" temporal aggregation pass through unchanged.
func TestDeaverageRunningMean_InstantUntouched(t *testing.T) {
	refTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	t2mEntries := []timedValues{
		makeEntry(refTime, refTime, refTime, 285),
		makeEntry(refTime, refTime.Add(time.Hour), refTime.Add(time.Hour), 286),
	}
	totPrEntries := []timedValues{
		makeEntry(refTime, refTime, refTime, 0.0001),
		makeEntry(refTime, refTime.Add(time.Hour), refTime.Add(time.Hour), 0.0002),
	}

	values := map[string][]timedValues{
		vars.VarT2m.Name:   t2mEntries,
		vars.VarTotPr.Name: totPrEntries,
	}
	deaverageRunningAggregates(values)

	require.Equal(t, float32(285), values[vars.VarT2m.Name][0].vals[0])
	require.Equal(t, float32(286), values[vars.VarT2m.Name][1].vals[0])
	require.Equal(t, float32(0.0001), values[vars.VarTotPr.Name][0].vals[0])
	require.Equal(t, float32(0.0002), values[vars.VarTotPr.Name][1].vals[0])
}

// TestDifferentiateRunningAccumulation_BasicTwoStep verifies that consecutive
// running totals are differenced into per-step totals.
func TestDifferentiateRunningAccumulation_BasicTwoStep(t *testing.T) {
	refTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// Two stacked running totals: 1.0 at +1h, 2.3 at +2h. Step 2 amount = 1.3.
	entries := []timedValues{
		makeEntry(refTime, refTime, refTime.Add(time.Hour), 1.0),
		makeEntry(refTime, refTime, refTime.Add(2*time.Hour), 2.3),
	}

	values := map[string][]timedValues{vars.VarTotPrec.Name: entries}
	deaverageRunningAggregates(values)

	require.InDelta(t, 1.0, float64(values[vars.VarTotPrec.Name][0].vals[0]), 1e-5,
		"first step is one hour wide, value stays unchanged")
	require.InDelta(t, 1.3, float64(values[vars.VarTotPrec.Name][1].vals[0]), 1e-5,
		"second step is differenced: 2.3 - 1.0 = 1.3")
}

// TestDifferentiateRunningAccumulation_NaNPropagation verifies that a NaN in
// either predecessor or current produces NaN in the output, while clean
// neighbouring locations are computed correctly.
func TestDifferentiateRunningAccumulation_NaNPropagation(t *testing.T) {
	refTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	nan := float32(math.NaN())

	entries := []timedValues{
		makeEntry(refTime, refTime, refTime.Add(time.Hour), 1.0, nan),
		makeEntry(refTime, refTime, refTime.Add(2*time.Hour), 2.3, 5.0),
	}

	values := map[string][]timedValues{vars.VarTotPrec.Name: entries}
	deaverageRunningAggregates(values)

	require.InDelta(t, 1.3, float64(values[vars.VarTotPrec.Name][1].vals[0]), 1e-5)
	require.True(t, math.IsNaN(float64(values[vars.VarTotPrec.Name][1].vals[1])),
		"NaN predecessor must propagate to the per-step value")
}
