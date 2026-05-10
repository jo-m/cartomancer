package roadclosures

import (
	"testing"

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

func TestNullString(t *testing.T) {
	ns := NullString("")
	require.False(t, ns.Valid)

	ns = NullString("hello")
	require.True(t, ns.Valid)
	require.Equal(t, "hello", ns.String)
}
