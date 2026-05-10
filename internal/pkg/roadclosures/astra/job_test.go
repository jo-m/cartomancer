package astra

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseDurationRange_Valid(t *testing.T) {
	start, end := parseDurationRange("19.11.2025 – 01.05.2026")
	require.True(t, start.Valid)
	require.True(t, end.Valid)
	require.Equal(t, 2025, start.Time.Year())
	require.Equal(t, time.November, start.Time.Month())
	require.Equal(t, 19, start.Time.Day())
	require.Equal(t, 2026, end.Time.Year())
	require.Equal(t, time.May, end.Time.Month())
	require.Equal(t, 1, end.Time.Day())
}

func TestParseDurationRange_NoSpaces(t *testing.T) {
	start, end := parseDurationRange("02.02.2026–05.06.2026")
	require.True(t, start.Valid)
	require.True(t, end.Valid)
	require.Equal(t, 2, start.Time.Day())
	require.Equal(t, 5, end.Time.Day())
}

func TestParseDurationRange_UntilFurtherNotice(t *testing.T) {
	start, end := parseDurationRange("until further notice")
	require.False(t, start.Valid)
	require.False(t, end.Valid)
}

func TestParseDurationRange_Empty(t *testing.T) {
	start, end := parseDurationRange("")
	require.False(t, start.Valid)
	require.False(t, end.Valid)
}

func TestParseDurationRange_UntilEndOf(t *testing.T) {
	start, end := parseDurationRange("until the end of 2027")
	require.False(t, start.Valid)
	require.False(t, end.Valid)
}
