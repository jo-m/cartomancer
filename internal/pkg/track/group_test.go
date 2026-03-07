package track

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// makeCells builds a Cells from a line of points for use in group tests.
func makeCells(t *testing.T, lat0, lon0, lat1, lon1 float64, n int) *Cells {
	t.Helper()
	pts := line(lat0, lon0, lat1, lon1, n, time.Now())
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypePlanned},
		points: pts,
	}
	c, err := NewCells(src, 7)
	require.NoError(t, err)
	return c
}

func TestGroup_EmptyInput(t *testing.T) {
	_, err := Group(nil, 0.5)
	require.Error(t, err)

	_, err = Group([]*Cells{}, 0.5)
	require.Error(t, err)
}

func TestGroup_SingleTrack(t *testing.T) {
	c := makeCells(t, 52.50, 13.40, 52.55, 13.45, 20)

	res, err := Group([]*Cells{c}, 0.5)
	require.NoError(t, err)
	require.Empty(t, res.Groups)
	require.Equal(t, []int{0}, res.NotMatched)
}

func TestGroup_IdenticalTracks(t *testing.T) {
	c1 := makeCells(t, 52.50, 13.40, 52.55, 13.45, 20)
	c2 := makeCells(t, 52.50, 13.40, 52.55, 13.45, 20)

	res, err := Group([]*Cells{c1, c2}, 0.5)
	require.NoError(t, err)
	require.Len(t, res.Groups, 1)
	require.Contains(t, res.Groups[0], 0)
	require.Contains(t, res.Groups[0], 1)
	require.Empty(t, res.NotMatched)
}

func TestGroup_DisjointTracks(t *testing.T) {
	// Two tracks in completely different locations.
	c1 := makeCells(t, 52.50, 13.40, 52.55, 13.45, 20)
	c2 := makeCells(t, 48.10, 11.50, 48.15, 11.55, 20)

	res, err := Group([]*Cells{c1, c2}, 0.5)
	require.NoError(t, err)
	require.Empty(t, res.Groups)
	require.ElementsMatch(t, []int{0, 1}, res.NotMatched)
}

func TestGroup_DifferentResolutions(t *testing.T) {
	c1 := makeCells(t, 52.50, 13.40, 52.55, 13.45, 20)
	c2 := &Cells{cells: c1.cells, nZeros: c1.nZeros, res: 5}

	_, err := Group([]*Cells{c1, c2}, 0.5)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolution")
}

func TestGroup_ThreeTracksPartialOverlap(t *testing.T) {
	// Tracks A and B share the same route; track C is elsewhere.
	a := makeCells(t, 52.50, 13.40, 52.55, 13.45, 20)
	b := makeCells(t, 52.50, 13.40, 52.55, 13.45, 20)
	c := makeCells(t, 48.10, 11.50, 48.15, 11.55, 20)

	res, err := Group([]*Cells{a, b, c}, 0.5)
	require.NoError(t, err)
	require.Len(t, res.Groups, 1)
	require.Contains(t, res.Groups[0], 0)
	require.Contains(t, res.Groups[0], 1)
	require.Equal(t, []int{2}, res.NotMatched)
}

func TestGroup_LowThresholdGroupsPartialOverlap(t *testing.T) {
	// Track A: X to Y. Track B: X to Y, then continues further to Z.
	// They share the entire first segment, so the shorter track has a high overlap ratio.
	a := makeCells(t, 52.50, 13.40, 52.55, 13.45, 30)
	b := makeCells(t, 52.50, 13.40, 52.60, 13.50, 60)

	// With a low threshold, the shared portion should be enough to group them.
	res, err := Group([]*Cells{a, b}, 0.1)
	require.NoError(t, err)
	require.Len(t, res.Groups, 1)
}

func TestGroup_HighThresholdRejectsPartialOverlap(t *testing.T) {
	// Same tracks as above, but a high threshold rejects them because the longer
	// track has a low ratio of shared edges.
	a := makeCells(t, 52.50, 13.40, 52.55, 13.45, 30)
	b := makeCells(t, 52.50, 13.40, 52.60, 13.50, 60)

	res, err := Group([]*Cells{a, b}, 0.99)
	require.NoError(t, err)
	require.Empty(t, res.Groups)
	require.ElementsMatch(t, []int{0, 1}, res.NotMatched)
}
