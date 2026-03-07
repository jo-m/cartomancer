package track

import (
	"iter"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mockSource implements TrackSource for testing.
type mockSource struct {
	meta   Metadata
	points []Point
}

func (m *mockSource) Metadata() Metadata { return m.meta }

func (m *mockSource) All() iter.Seq[Point] {
	return func(yield func(Point) bool) {
		for _, p := range m.points {
			if !yield(p) {
				return
			}
		}
	}
}

// line returns n evenly spaced points between two lat/lon pairs.
// Timestamps are spaced 1 second apart starting from t0.
func line(lat0, lon0, lat1, lon1 float64, n int, t0 time.Time) []Point {
	pts := make([]Point, n)
	for i := range n {
		x := float64(i) / float64(n-1)
		pts[i] = Point{
			Time: t0.Add(time.Duration(i) * time.Second),
			Lat:  lat0 + (lat1-lat0)*x,
			Lon:  lon0 + (lon1-lon0)*x,
		}
	}
	return pts
}

func TestCheckAndSplit_TooFewPoints(t *testing.T) {
	_, err := checkAndSplit(nil, false)
	require.Error(t, err)

	_, err = checkAndSplit([]Point{{}, {}}, false)
	require.NoError(t, err)

	_, err = checkAndSplit([]Point{{}}, false)
	require.Error(t, err)
}

func TestCheckAndSplit_UnorderedTimestamps(t *testing.T) {
	t0 := time.Now()
	pts := []Point{
		{Time: t0.Add(1 * time.Second), Lat: 52.0, Lon: 13.0},
		{Time: t0, Lat: 52.001, Lon: 13.001},
	}
	_, err := checkAndSplit(pts, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timestamps")
}

func TestCheckAndSplit_PlannedNoSplit(t *testing.T) {
	t0 := time.Now()
	// Two points far apart: planned tracks should not split.
	pts := []Point{
		{Time: t0, Lat: 52.0, Lon: 13.0},
		{Time: t0.Add(time.Second), Lat: 53.0, Lon: 14.0},
	}
	segs, err := checkAndSplit(pts, false)
	require.NoError(t, err)
	require.Len(t, segs, 1)
	require.Len(t, segs[0], 2)
}

func TestCheckAndSplit_RecordedSplitsOnGap(t *testing.T) {
	t0 := time.Now()
	// Three close points, then a far-away point (>200m), then two more close points.
	pts := []Point{
		{Time: t0, Lat: 52.520, Lon: 13.405},
		{Time: t0.Add(1 * time.Second), Lat: 52.5201, Lon: 13.4051},
		{Time: t0.Add(2 * time.Second), Lat: 52.5202, Lon: 13.4052},
		// Jump ~11 km north.
		{Time: t0.Add(3 * time.Second), Lat: 52.620, Lon: 13.405},
		{Time: t0.Add(4 * time.Second), Lat: 52.6201, Lon: 13.4051},
	}
	segs, err := checkAndSplit(pts, true)
	require.NoError(t, err)
	require.Len(t, segs, 2, "should split into two segments on large gap")
	require.Len(t, segs[0], 3)
	require.Len(t, segs[1], 2)
}

func TestCheckAndSplit_RecordedDiscardsSinglePointSegment(t *testing.T) {
	t0 := time.Now()
	// Two close points, then a gap, then a single point (no pair), then another gap, then two close.
	pts := []Point{
		{Time: t0, Lat: 52.520, Lon: 13.405},
		{Time: t0.Add(1 * time.Second), Lat: 52.5201, Lon: 13.4051},
		// Gap.
		{Time: t0.Add(2 * time.Second), Lat: 53.0, Lon: 14.0},
		// Gap.
		{Time: t0.Add(3 * time.Second), Lat: 54.0, Lon: 15.0},
		{Time: t0.Add(4 * time.Second), Lat: 54.0001, Lon: 15.0001},
	}
	segs, err := checkAndSplit(pts, true)
	require.NoError(t, err)
	// The single-point segment (53.0) should be discarded.
	require.Len(t, segs, 2)
}

func TestCheckAndSplit_RecordedClosePointsNoSplit(t *testing.T) {
	t0 := time.Now()
	// All points within 200m of each other.
	pts := line(52.520, 13.405, 52.521, 13.406, 5, t0)
	segs, err := checkAndSplit(pts, true)
	require.NoError(t, err)
	require.Len(t, segs, 1)
	require.Len(t, segs[0], 5)
}

func TestInterpolatePoints_DeduplicatesSameCell(t *testing.T) {
	// Two points very close together should map to the same cell.
	t0 := time.Now()
	pts := []Point{
		{Time: t0, Lat: 52.520, Lon: 13.405},
		{Time: t0.Add(time.Second), Lat: 52.52000001, Lon: 13.40500001},
	}
	cells := interpolatePoints(pts, 5)
	require.Len(t, cells, 1, "nearby points should collapse to one cell")
}

func TestInterpolatePoints_ProducesMultipleCells(t *testing.T) {
	// Points far enough apart to span multiple H3 cells at resolution 7.
	t0 := time.Now()
	pts := line(52.50, 13.40, 52.55, 13.45, 10, t0)
	cells := interpolatePoints(pts, 7)
	require.Greater(t, len(cells), 1, "distant points should produce multiple cells")

	// No consecutive duplicates.
	for i := 1; i < len(cells); i++ {
		require.NotEqual(t, cells[i-1], cells[i], "consecutive cells must differ")
	}
}

func TestNewCells_PlannedTrack(t *testing.T) {
	t0 := time.Now()
	pts := line(52.50, 13.40, 52.55, 13.45, 20, t0)
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypePlanned},
		points: pts,
	}

	cells, err := NewCells(src, 7)
	require.NoError(t, err)
	require.Equal(t, 1, cells.NSegments())
	require.Greater(t, cells.NCells(), 1)
}

func TestNewCells_RecordedTrackWithSegments(t *testing.T) {
	t0 := time.Now()
	seg1 := line(52.520, 13.405, 52.521, 13.406, 5, t0)
	// Gap > 200m to force a segment split.
	seg2 := line(52.620, 13.505, 52.621, 13.506, 5, t0.Add(10*time.Second))
	pts := append(seg1, seg2...)

	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypeRecorded},
		points: pts,
	}

	cells, err := NewCells(src, 7)
	require.NoError(t, err)
	require.Equal(t, 2, cells.NSegments())
	require.Greater(t, cells.NCells(), 0)

	// Verify zero separator exists in internal slice.
	require.True(t, slices.Contains(cells.cells, 0), "segment separator (zero) should be present")
}

func TestNewCells_TooFewPoints(t *testing.T) {
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypePlanned},
		points: []Point{{Time: time.Now()}},
	}
	_, err := NewCells(src, 7)
	require.Error(t, err)
}

func TestNewCells_RecordedNoSplitWhenClose(t *testing.T) {
	t0 := time.Now()
	pts := line(52.520, 13.405, 52.521, 13.406, 10, t0)
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypeRecorded},
		points: pts,
	}

	cells, err := NewCells(src, 7)
	require.NoError(t, err)
	require.Equal(t, 1, cells.NSegments(), "close points should not split")
}

func TestCellsRoundTrip(t *testing.T) {
	t0 := time.Now()
	pts := line(52.50, 13.40, 52.55, 13.45, 20, t0)
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypePlanned},
		points: pts,
	}

	original, err := NewCells(src, 7)
	require.NoError(t, err)

	data, err := original.CellsToBytes()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	restored, err := CellsFromBytes(data)
	require.NoError(t, err)

	require.Equal(t, original.cells, restored.cells)
	require.Equal(t, original.nZeros, restored.nZeros)
	require.Equal(t, original.res, restored.res)
}

func TestCellsRoundTripWithSegments(t *testing.T) {
	t0 := time.Now()
	seg1 := line(52.520, 13.405, 52.521, 13.406, 5, t0)
	seg2 := line(52.620, 13.505, 52.621, 13.506, 5, t0.Add(10*time.Second))
	pts := append(seg1, seg2...)
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypeRecorded},
		points: pts,
	}

	original, err := NewCells(src, 7)
	require.NoError(t, err)
	require.Equal(t, 2, original.NSegments())

	data, err := original.CellsToBytes()
	require.NoError(t, err)

	restored, err := CellsFromBytes(data)
	require.NoError(t, err)
	require.Equal(t, original.NSegments(), restored.NSegments())
	require.Equal(t, original.NCells(), restored.NCells())
}

func TestCellsFromBytes_InvalidData(t *testing.T) {
	_, err := CellsFromBytes([]byte("garbage"))
	require.Error(t, err)
}

func BenchmarkNewCells_SmallPlanned(b *testing.B) {
	t0 := time.Now()
	pts := line(52.50, 13.40, 52.55, 13.45, 50, t0)
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypePlanned},
		points: pts,
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := NewCells(src, 7)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewCells_LargePlanned(b *testing.B) {
	t0 := time.Now()
	pts := line(52.50, 13.40, 53.50, 14.40, 5000, t0)
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypePlanned},
		points: pts,
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := NewCells(src, 7)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewCells_RecordedWithSegments(b *testing.B) {
	t0 := time.Now()
	seg1 := line(52.520, 13.405, 52.530, 13.415, 500, t0)
	seg2 := line(52.620, 13.505, 52.630, 13.515, 500, t0.Add(10*time.Minute))
	seg3 := line(52.720, 13.605, 52.730, 13.615, 500, t0.Add(20*time.Minute))
	pts := append(append(seg1, seg2...), seg3...)
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypeRecorded},
		points: pts,
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := NewCells(src, 7)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewCells_HighResolution(b *testing.B) {
	t0 := time.Now()
	pts := line(52.50, 13.40, 52.55, 13.45, 500, t0)
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypePlanned},
		points: pts,
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := NewCells(src, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestNCells_ExcludesZeros(t *testing.T) {
	t0 := time.Now()
	seg1 := line(52.520, 13.405, 52.521, 13.406, 5, t0)
	seg2 := line(52.620, 13.505, 52.621, 13.506, 5, t0.Add(10*time.Second))
	pts := append(seg1, seg2...)
	src := &mockSource{
		meta:   Metadata{TrackType: TrackTypeRecorded},
		points: pts,
	}

	cells, err := NewCells(src, 7)
	require.NoError(t, err)

	// NCells should equal total slice length minus the number of zero separators.
	require.Equal(t, len(cells.cells)-cells.nZeros, cells.NCells())
}
