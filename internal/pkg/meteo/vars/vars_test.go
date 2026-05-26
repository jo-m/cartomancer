package vars

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsRunningMeanFromReferenceTime verifies that the helper correctly
// identifies variables that ICON-CH1-EPS publishes as running means from the
// model reference time.
func TestIsRunningMeanFromReferenceTime(t *testing.T) {
	require.True(t, VarAswdirS.IsRunningMeanFromReferenceTime(),
		"ASWDIR_S is published as a running mean from reference time")
	require.True(t, VarAswdifdS.IsRunningMeanFromReferenceTime(),
		"ASWDIFD_S is published as a running mean from reference time")
	require.True(t, VarAsobS.IsRunningMeanFromReferenceTime(),
		"ASOB_S is published as a running mean from reference time")

	require.False(t, VarT2m.IsRunningMeanFromReferenceTime(),
		"T_2M is instantaneous, not a running mean")
	require.False(t, VarTotPr.IsRunningMeanFromReferenceTime(),
		"TOT_PR is instantaneous, not a running mean")
	require.False(t, VarTotPrec.IsRunningMeanFromReferenceTime(),
		"TOT_PREC is an accumulation, not a running mean")
}

// TestIsRunningAccumulationFromReferenceTime verifies that the helper
// correctly identifies variables that ICON-CH1-EPS publishes as running
// totals from the model reference time.
func TestIsRunningAccumulationFromReferenceTime(t *testing.T) {
	require.True(t, VarTotPrec.IsRunningAccumulationFromReferenceTime(),
		"TOT_PREC is published as a running accumulation from reference time")

	require.False(t, VarT2m.IsRunningAccumulationFromReferenceTime(),
		"T_2M is instantaneous, not a running accumulation")
	require.False(t, VarTotPr.IsRunningAccumulationFromReferenceTime(),
		"TOT_PR is instantaneous, not a running accumulation")
	require.False(t, VarAswdirS.IsRunningAccumulationFromReferenceTime(),
		"ASWDIR_S is a running mean, not a running accumulation")
}

// TestByName verifies the name-based [Variable] lookup against entries in the
// generated [Variables] list.
func TestByName(t *testing.T) {
	v, ok := ByName("T_2M")
	require.True(t, ok)
	require.Equal(t, VarT2m, v)

	v, ok = ByName("ASWDIR_S")
	require.True(t, ok)
	require.Equal(t, VarAswdirS, v)

	_, ok = ByName("NOT_A_REAL_VAR")
	require.False(t, ok)
}
