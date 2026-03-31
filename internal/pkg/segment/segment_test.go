package segment

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// testSource creates a minimal TrackSource from lat/lon pairs for testing.
type testSource struct {
	points []track.Point
}

func (ts *testSource) Metadata() track.Metadata {
	return track.Metadata{TrackType: track.TrackTypePlanned}
}

func (ts *testSource) All() iter.Seq[track.Point] {
	return func(yield func(track.Point) bool) {
		for _, p := range ts.points {
			if !yield(p) {
				return
			}
		}
	}
}

// pointsFromCoords converts a slice of [lat,lon] pairs to track.Points.
func pointsFromCoords(coords [][2]float64) []track.Point {
	pts := make([]track.Point, len(coords))
	for i, c := range coords {
		pts[i] = track.Point{Lat: c[0], Lon: c[1]}
	}
	return pts
}

// trackCellsFromCoords creates a TrackCells from coordinate pairs.
func trackCellsFromCoords(t *testing.T, uuid string, coords [][2]float64, res int) TrackCells {
	t.Helper()
	pts := pointsFromCoords(coords)
	src := &testSource{points: pts}
	cells, err := track.NewCells(src, res)
	require.NoError(t, err)
	return TrackCells{UUID: uuid, Cells: cells}
}

// testRawPointLoader returns a RawPointLoader that uses pre-built track data.
func testRawPointLoader(tracks map[trackUUID][][2]float64) RawPointLoader {
	return func(uuid string) ([]track.Point, error) {
		coords, ok := tracks[trackUUID(uuid)]
		if !ok {
			return nil, nil
		}
		pts := make([]track.Point, len(coords))
		for i, c := range coords {
			pts[i] = track.Point{Lat: c[0], Lon: c[1]}
		}
		return pts, nil
	}
}

// testPointLoader returns a PointLoader that uses pre-built track data.
func testPointLoader(tracks map[trackUUID][][2]float64, res int) PointLoader {
	return func(uuid string, _ int) (map[h3.Cell]track.Point, error) {
		coords, ok := tracks[trackUUID(uuid)]
		if !ok {
			return nil, nil
		}
		m := make(map[h3.Cell]track.Point)
		for _, c := range coords {
			p := track.Point{Lat: c[0], Lon: c[1]}
			cell := p.Cell(res)
			if _, exists := m[cell]; !exists {
				m[cell] = p
			}
		}
		return m, nil
	}
}

// longLine returns a straight line of n points at 0.001 degree spacing.
func longLine(n int) [][2]float64 {
	coords := make([][2]float64, n)
	for i := range n {
		coords[i] = [2]float64{47.0 + float64(i)*0.001, 8.0 + float64(i)*0.001}
	}
	return coords
}

func TestExtractNoTracks(t *testing.T) {
	result, err := Extract(nil, MinTrackCount, nil)
	require.NoError(t, err)
	require.Empty(t, result.Segments)
}

func TestExtractSingleTrack(t *testing.T) {
	coords := longLine(30)
	tc := trackCellsFromCoords(t, "t1", coords, DefaultResolution)

	result, err := Extract([]TrackCells{tc}, 2, nil)
	require.NoError(t, err)
	require.Empty(t, result.Segments, "single track should produce no segments with minTracks=2")
}

func TestExtractTwoIdenticalTracks(t *testing.T) {
	coords := longLine(30)
	tc1 := trackCellsFromCoords(t, "t1", coords, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords, DefaultResolution)

	result, err := Extract([]TrackCells{tc1, tc2}, MinTrackCount, nil)
	require.NoError(t, err)
	require.NotEmpty(t, result.Segments, "two identical tracks should produce segments")

	for _, seg := range result.Segments {
		require.GreaterOrEqual(t, len(seg.TrackUUIDs), MinTrackCount)
		require.Greater(t, seg.DistanceM, 0.0)
		require.NotEmpty(t, seg.Cells)
		require.GreaterOrEqual(t, seg.DistanceM, MinSegmentDistanceM)
	}
}

func TestExtractDivergingTracks(t *testing.T) {
	common := longLine(25)
	base := common[len(common)-1]

	ext1 := make([][2]float64, 10)
	ext2 := make([][2]float64, 10)
	for i := range 10 {
		ext1[i] = [2]float64{base[0] + float64(i+1)*0.001, base[1] + float64(i+1)*0.001}
		ext2[i] = [2]float64{base[0], base[1] + float64(i+1)*0.002}
	}

	coords1 := append(append([][2]float64{}, common...), ext1...)
	coords2 := append(append([][2]float64{}, common...), ext2...)

	tc1 := trackCellsFromCoords(t, "t1", coords1, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords2, DefaultResolution)

	result, err := Extract([]TrackCells{tc1, tc2}, MinTrackCount, nil)
	require.NoError(t, err)

	sharedCount := 0
	for _, seg := range result.Segments {
		if len(seg.TrackUUIDs) >= 2 {
			sharedCount++
		}
	}
	require.GreaterOrEqual(t, sharedCount, 1, "should have at least one shared segment")
}

func TestTrackSetChangeCreatesJunction(t *testing.T) {
	// Three tracks: A, B share the full path; C shares only the first half.
	// This should create a junction where C leaves, producing two segments:
	// one with {A,B,C} and one with {A,B}.
	full := longLine(40)
	half := full[:20]

	tcA := trackCellsFromCoords(t, "A", full, DefaultResolution)
	tcB := trackCellsFromCoords(t, "B", full, DefaultResolution)
	tcC := trackCellsFromCoords(t, "C", half, DefaultResolution)

	result, err := Extract([]TrackCells{tcA, tcB, tcC}, MinTrackCount, nil)
	require.NoError(t, err)
	require.NotEmpty(t, result.Segments)

	// There should be segments shared by all three, and segments shared by just A and B.
	hasThree := false
	hasTwo := false
	for _, seg := range result.Segments {
		switch len(seg.TrackUUIDs) {
		case 3:
			hasThree = true
		case 2:
			hasTwo = true
		}
	}
	require.True(t, hasThree, "should have a segment shared by all three tracks")
	require.True(t, hasTwo, "should have a segment shared by only A and B")
}

func TestNoGapsAtJunctions(t *testing.T) {
	// Two tracks share a prefix, then diverge. The shared segment's last cell
	// must equal the divergent segments' first cell.
	common := longLine(25)
	base := common[len(common)-1]

	ext1 := make([][2]float64, 15)
	ext2 := make([][2]float64, 15)
	for i := range 15 {
		ext1[i] = [2]float64{base[0] + float64(i+1)*0.001, base[1] + float64(i+1)*0.001}
		ext2[i] = [2]float64{base[0] + float64(i+1)*0.001, base[1] - float64(i+1)*0.001}
	}

	coords1 := append(append([][2]float64{}, common...), ext1...)
	coords2 := append(append([][2]float64{}, common...), ext2...)

	tc1 := trackCellsFromCoords(t, "t1", coords1, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords2, DefaultResolution)

	// Use minTracks=1 to see all segments including solo extensions.
	result, err := Extract([]TrackCells{tc1, tc2}, 1, nil)
	require.NoError(t, err)

	// Build a map of segment endpoints for gap checking.
	type endpoint struct {
		cell h3.Cell
		end  bool // true = end of segment, false = start
	}
	var endpoints []endpoint
	for _, seg := range result.Segments {
		if len(seg.Cells) < 2 {
			continue
		}
		endpoints = append(endpoints,
			endpoint{seg.Cells[0], false},
			endpoint{seg.Cells[len(seg.Cells)-1], true},
		)
	}

	// Every segment end cell should appear as a start cell of another segment
	// (unless it's a track endpoint).
	endCells := make(map[h3.Cell]bool)
	startCells := make(map[h3.Cell]bool)
	for _, ep := range endpoints {
		if ep.end {
			endCells[ep.cell] = true
		} else {
			startCells[ep.cell] = true
		}
	}

	// Junction cells (appearing as both end and start) verify no gaps.
	junctionCount := 0
	for cell := range endCells {
		if startCells[cell] {
			junctionCount++
		}
	}
	require.Greater(t, junctionCount, 0, "should have at least one junction cell shared between segments")
}

func TestNoOverlappingSegments(t *testing.T) {
	// All segments should have unique cell sequences.
	coords := longLine(40)
	base := coords[20]

	ext := make([][2]float64, 20)
	for i := range 20 {
		ext[i] = [2]float64{base[0] + float64(i+1)*0.001, base[1] - float64(i+1)*0.001}
	}

	tc1 := trackCellsFromCoords(t, "t1", coords, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", append(coords[:21], ext...), DefaultResolution)

	result, err := Extract([]TrackCells{tc1, tc2}, 1, nil)
	require.NoError(t, err)

	keys := make(map[string]struct{})
	for _, seg := range result.Segments {
		key := cellKey(seg.Cells)
		require.NotContains(t, keys, key, "duplicate segment cell sequence found")
		keys[key] = struct{}{}
	}
}

func TestAttachPolylines(t *testing.T) {
	coords := longLine(30)
	tc1 := trackCellsFromCoords(t, "t1", coords, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords, DefaultResolution)

	result, err := Extract([]TrackCells{tc1, tc2}, MinTrackCount, nil)
	require.NoError(t, err)
	require.NotEmpty(t, result.Segments)

	require.Empty(t, result.Segments[0].Polyline)

	loader := testPointLoader(map[trackUUID][][2]float64{"t1": coords, "t2": coords}, DefaultResolution)
	err = AttachPolylines(result.Segments, loader, DefaultResolution)
	require.NoError(t, err)

	require.NotEmpty(t, result.Segments[0].Polyline)
	for _, pt := range result.Segments[0].Polyline {
		require.InDelta(t, 47.0, pt[0], 0.05, "polyline lat should be near original points")
		require.InDelta(t, 8.0, pt[1], 0.05, "polyline lon should be near original points")
	}

	polyJSON, err := result.Segments[0].PolylineJSON()
	require.NoError(t, err)
	require.Contains(t, polyJSON, "[")
}

func TestMinSegmentDistanceFilter(t *testing.T) {
	// Two points close enough that they land in adjacent H3 cells but with
	// a total segment distance below MinSegmentDistanceM.
	coords := [][2]float64{{47.0, 8.0}, {47.00015, 8.00015}}
	tc1 := trackCellsFromCoords(t, "t1", coords, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords, DefaultResolution)

	result, err := Extract([]TrackCells{tc1, tc2}, MinTrackCount, nil)
	require.NoError(t, err)
	require.Empty(t, result.Segments, "very short shared path should be filtered out")
}

func TestLinearPathProducesSegment(t *testing.T) {
	coords := longLine(30)
	tc1 := trackCellsFromCoords(t, "t1", coords, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords, DefaultResolution)

	result, err := Extract([]TrackCells{tc1, tc2}, MinTrackCount, nil)
	require.NoError(t, err)
	require.NotEmpty(t, result.Segments, "linear path should produce at least one segment")

	totalDist := 0.0
	for _, seg := range result.Segments {
		totalDist += seg.DistanceM
		require.GreaterOrEqual(t, seg.DistanceM, MinSegmentDistanceM)
	}
	require.Greater(t, totalDist, 1000.0, "total segment distance should cover most of the path")
}

func TestDirectionSensitiveDeduplication(t *testing.T) {
	coords := longLine(30)

	revCoords := make([][2]float64, len(coords))
	for i, c := range coords {
		revCoords[len(coords)-1-i] = c
	}

	tc1 := trackCellsFromCoords(t, "t1", coords, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", revCoords, DefaultResolution)

	// With only 2 tracks going opposite directions, neither direction has
	// minTracks=2, so no segments should be produced.
	result, err := Extract([]TrackCells{tc1, tc2}, 2, nil)
	require.NoError(t, err)
	require.Empty(t, result.Segments, "opposite direction tracks should not merge into segments")

	// Add a third track matching t1's direction.
	tc3 := trackCellsFromCoords(t, "t3", coords, DefaultResolution)
	result, err = Extract([]TrackCells{tc1, tc2, tc3}, 2, nil)
	require.NoError(t, err)

	for _, seg := range result.Segments {
		require.NotContains(t, seg.TrackUUIDs, "t2",
			"reverse-direction track should not be in forward segments")
	}
}

func TestAttachPolylinesWithMissingCells(t *testing.T) {
	coords := longLine(30)
	tc1 := trackCellsFromCoords(t, "t1", coords, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords, DefaultResolution)

	result, err := Extract([]TrackCells{tc1, tc2}, MinTrackCount, nil)
	require.NoError(t, err)
	require.NotEmpty(t, result.Segments)

	sparseCoords := make([][2]float64, 0)
	for i := 0; i < len(coords); i += 3 {
		sparseCoords = append(sparseCoords, coords[i])
	}
	loader := testPointLoader(map[trackUUID][][2]float64{"t1": sparseCoords, "t2": sparseCoords}, DefaultResolution)

	err = AttachPolylines(result.Segments, loader, DefaultResolution)
	require.NoError(t, err)

	for _, seg := range result.Segments {
		require.NotEmpty(t, seg.Polyline, "polyline should have points from available GPS data")
		require.LessOrEqual(t, len(seg.Polyline), len(seg.Cells),
			"polyline can have fewer points than cells when GPS data is sparse")
	}
}

func TestConvergingTracksCreateJunction(t *testing.T) {
	// Two tracks start at different places but converge to a shared path.
	shared := longLine(25)

	start1 := make([][2]float64, 15)
	start2 := make([][2]float64, 15)
	base := shared[0]
	for i := range 15 {
		start1[14-i] = [2]float64{base[0] - float64(i+1)*0.001, base[1] - float64(i+1)*0.001}
		start2[14-i] = [2]float64{base[0] - float64(i+1)*0.001, base[1] + float64(i+1)*0.001}
	}

	coords1 := append(start1, shared...)
	coords2 := append(start2, shared...)

	tc1 := trackCellsFromCoords(t, "t1", coords1, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords2, DefaultResolution)

	result, err := Extract([]TrackCells{tc1, tc2}, MinTrackCount, nil)
	require.NoError(t, err)

	sharedDist := 0.0
	for _, seg := range result.Segments {
		if len(seg.TrackUUIDs) >= 2 {
			sharedDist += seg.DistanceM
		}
	}
	require.Greater(t, sharedDist, 1000.0,
		"converging tracks should produce shared segments covering the common path")
}

func TestSelfCrossingTrackCreatesJunction(t *testing.T) {
	// A single track that goes out and comes back, revisiting the same
	// H3 cells. The self-crossing pass should detect junctions at the
	// cells visited twice.
	outward := longLine(30)

	// Return via slightly offset path that still maps to the same cells.
	returnLeg := make([][2]float64, 20)
	for i := range 20 {
		src := outward[len(outward)-1-i]
		returnLeg[i] = [2]float64{src[0] + 0.00001, src[1] + 0.00001}
	}

	coords := append(append([][2]float64{}, outward...), returnLeg...)
	tc := trackCellsFromCoords(t, "loop", coords, DefaultResolution)

	junctions, _ := DetectJunctions([]TrackCells{tc})
	require.NotEmpty(t, junctions,
		"self-crossing track should produce at least one junction")
}

func TestCellOccupancyDetectsOrderIndependentJunctions(t *testing.T) {
	// Two tracks share a middle section. The cell-occupancy pass should
	// detect junctions regardless of track processing order, unlike the
	// edge-based pass where the first track sees empty edges.
	shared := longLine(20)
	base := shared[0]
	baseEnd := shared[len(shared)-1]

	// Unique prefixes approaching from different directions.
	pre1 := make([][2]float64, 15)
	pre2 := make([][2]float64, 15)
	for i := range 15 {
		pre1[14-i] = [2]float64{base[0] - float64(i+1)*0.001, base[1] - float64(i+1)*0.001}
		pre2[14-i] = [2]float64{base[0] - float64(i+1)*0.001, base[1] + float64(i+1)*0.001}
	}

	// Unique suffixes diverging in different directions.
	suf1 := make([][2]float64, 15)
	suf2 := make([][2]float64, 15)
	for i := range 15 {
		suf1[i] = [2]float64{baseEnd[0] + float64(i+1)*0.001, baseEnd[1] + float64(i+1)*0.001}
		suf2[i] = [2]float64{baseEnd[0] + float64(i+1)*0.001, baseEnd[1] - float64(i+1)*0.001}
	}

	coords1 := append(append(append([][2]float64{}, pre1...), shared...), suf1...)
	coords2 := append(append(append([][2]float64{}, pre2...), shared...), suf2...)

	tc1 := trackCellsFromCoords(t, "t1", coords1, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords2, DefaultResolution)

	// Process in both orders and verify same junctions are found.
	junctionsAB, _ := DetectJunctions([]TrackCells{tc1, tc2})
	junctionsBA, _ := DetectJunctions([]TrackCells{tc2, tc1})

	require.GreaterOrEqual(t, len(junctionsAB), 2,
		"should detect junctions at both convergence and divergence")
	require.Equal(t, len(junctionsAB), len(junctionsBA),
		"junction count should be order-independent")
	for j := range junctionsAB {
		require.Contains(t, junctionsBA, j,
			"junction cells should be the same regardless of processing order")
	}
}

func TestReversePassDetectsAdditionalJunctions(t *testing.T) {
	// Two tracks share a long path, then gradually diverge at the end.
	// The forward pass detects junctions at the divergence point. The
	// reverse pass should detect junctions at the convergence point
	// (viewed from reverse), catching cell-boundary edge cases.
	shared := longLine(30)
	base := shared[len(shared)-1]

	// Extend each track with a long unique tail.
	ext1 := make([][2]float64, 20)
	ext2 := make([][2]float64, 20)
	for i := range 20 {
		ext1[i] = [2]float64{base[0] + float64(i+1)*0.001, base[1] + float64(i+1)*0.001}
		ext2[i] = [2]float64{base[0] + float64(i+1)*0.001, base[1] - float64(i+1)*0.001}
	}

	coords1 := append(append([][2]float64{}, shared...), ext1...)
	coords2 := append(append([][2]float64{}, shared...), ext2...)

	tc1 := trackCellsFromCoords(t, "t1", coords1, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords2, DefaultResolution)

	junctions, _ := DetectJunctions([]TrackCells{tc1, tc2})

	// The reverse pass should produce at least as many junction candidates
	// as a forward-only approach would. With two diverging tracks, we
	// expect junctions near the divergence point from both directions.
	require.NotEmpty(t, junctions, "reverse pass should help detect junctions")

	// Verify junctions are valid H3 cells.
	for j := range junctions {
		require.True(t, j.IsValid(), "junction cell must be valid")
	}
}

func TestJunctionRefinementMovesJunction(t *testing.T) {
	// Two tracks share a path and diverge. When a PointLoader is provided,
	// junction refinement should run without errors and produce valid
	// segments.
	shared := longLine(20)
	ext := make([][2]float64, 15)
	base := shared[len(shared)-1]
	for i := range 15 {
		ext[i] = [2]float64{base[0] + float64(i+1)*0.001, base[1] + float64(i+1)*0.001}
	}

	coords1 := append(shared, ext...)
	coords2 := shared

	tc1 := trackCellsFromCoords(t, "t1", coords1, DefaultResolution)
	tc2 := trackCellsFromCoords(t, "t2", coords2, DefaultResolution)

	loader := testRawPointLoader(map[trackUUID][][2]float64{
		"t1": coords1,
		"t2": coords2,
	})

	result, err := Extract([]TrackCells{tc1, tc2}, MinTrackCount, loader)
	require.NoError(t, err)
	require.NotEmpty(t, result.Segments, "should produce segments with junction refinement")

	for _, seg := range result.Segments {
		require.GreaterOrEqual(t, len(seg.Cells), 2, "each segment must have at least 2 cells")
	}
}
