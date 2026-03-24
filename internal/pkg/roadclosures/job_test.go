package roadclosures

import (
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"
	"github.com/uber/h3-go/v4"
)

func TestGeometryCells_Point(t *testing.T) {
	pt := orb.Point{8.5, 47.3}
	cells := geometryCells(pt, CellResolution)
	require.Len(t, cells, 1)
}

func TestGeometryCells_LineString(t *testing.T) {
	// A short line segment that should cover at least 2 cells.
	ls := orb.LineString{
		{8.5, 47.3},
		{8.52, 47.32},
	}
	cells := geometryCells(ls, CellResolution)
	require.Greater(t, len(cells), 1, "line string should cover multiple cells")
}

func TestGeometryCells_MultiLineString(t *testing.T) {
	mls := orb.MultiLineString{
		{{8.5, 47.3}, {8.52, 47.32}},
		{{9.0, 46.5}, {9.02, 46.52}},
	}
	cells := geometryCells(mls, CellResolution)
	require.Greater(t, len(cells), 2, "two line strings should cover many cells")
}

func TestGeometryCells_NilGeometry(t *testing.T) {
	cells := geometryCells(nil, CellResolution)
	require.Empty(t, cells)
}

func TestAddPoints_Interpolation(t *testing.T) {
	// Two points far apart should produce intermediate cells via interpolation.
	pts := []orb.Point{
		{8.0, 47.0},
		{8.1, 47.1},
	}
	cells := make(map[h3.Cell]struct{})
	addPoints(cells, pts, CellResolution)
	require.Greater(t, len(cells), 2, "interpolation should produce intermediate cells")
}

func TestParseDurationRange_Valid(t *testing.T) {
	start, end := parseDurationRange("19.11.2025 \u2013 01.05.2026")
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
	start, end := parseDurationRange("02.02.2026\u201305.06.2026")
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

func TestNullString(t *testing.T) {
	ns := nullString("")
	require.False(t, ns.Valid)

	ns = nullString("hello")
	require.True(t, ns.Valid)
	require.Equal(t, "hello", ns.String)
}
