package track

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/uber/h3-go/v4"
)

// Cells is a [Track] indexed to H3. Always contains at least one cell.
// Use [NewCells] to construct; it enforces the minimum-cell invariant.
type Cells struct {
	// Ordered list of H3 cells forming a track.
	// A track consists of one or more segments.
	// Segments are sequences of adjacent cells, separated by 0.
	// If a track has location discontinuities, it will be split into multiple segments.
	cells  []h3.Cell
	nZeros int
	// Resolution at which cells were created.
	res int
}

// NSegments returns the number of segments in the track.
func (c *Cells) NSegments() int {
	return c.nZeros + 1
}

// NCells returns the number of H3 cells in the track, excluding segment separators.
func (c *Cells) NCells() int {
	return len(c.cells) - c.nZeros
}

// NEdges returns the number of directed edges in the track.
// Each segment of N cells contributes N-1 edges.
func (c *Cells) NEdges() int {
	return c.NCells() - c.NSegments()
}

const (
	// maxPointDistRecordedM is the maximum allowed distance in meters between consecutive
	// points in a recorded track before a new segment is started.
	maxPointDistRecordedM = 200.
)

// checkAndSplit validates points and splits them into segments.
// For recorded tracks, a new segment is started whenever consecutive points are more than
// maxPointDistRecordedM apart. Segments with fewer than two points are discarded.
// Returns sub-slices of points; no new point storage is allocated.
// Returns an error if there are fewer than two points or timestamps are not ordered.
func checkAndSplit(points []Point, recorded bool) ([][]Point, error) {
	if len(points) < 2 {
		return nil, fmt.Errorf("not enough points")
	}

	segStart := 0
	var segs [][]Point

	for i, p1 := range points[1:] {
		p0 := points[i]

		if p1.Time.Before(p0.Time) {
			return nil, fmt.Errorf("timestamps are not ordered")
		}

		if recorded {
			d := p1.MetersTo(&p0)
			if d > maxPointDistRecordedM {
				if seg := points[segStart : i+1]; len(seg) > 1 {
					segs = append(segs, seg)
				}
				segStart = i + 1
			}
		}
	}

	// Flush the last segment.
	if seg := points[segStart:]; len(seg) > 1 {
		segs = append(segs, seg)
	}

	return segs, nil
}

// interpolatePoints converts a sequence of points into a deduplicated list of H3 cells
// at the given resolution. Points are interpolated at sub-cell granularity (half the
// average hexagon edge length) to avoid skipping cells on straight segments.
func interpolatePoints(pts []Point, resolution int) []h3.Cell {
	ret := []h3.Cell{pts[0].Cell(resolution)}

	edgeLenM, err := h3.HexagonEdgeLengthAvgM(resolution)
	if err != nil {
		panic(err)
	}
	interpolateMeters := edgeLenM / 2

	for i, p1 := range pts[1:] {
		p0 := pts[i]

		distM := p0.MetersTo(&p1)
		steps := int(distM/interpolateMeters + 1)
		for j := range steps {
			x := float64(j) / float64(steps)
			interp := p0.Interpolate(&p1, x)
			cell := interp.Cell(resolution)
			if ret[len(ret)-1] != cell {
				ret = append(ret, cell)
			}
		}
	}

	// Add the last cell.
	cell := pts[len(pts)-1].Cell(resolution)
	if ret[len(ret)-1] != cell {
		ret = append(ret, cell)
	}

	return ret
}

// NewCells builds a Cells index from a TrackSource at the given H3 resolution.
// Points are validated, split into segments on discontinuities, and interpolated
// into H3 cells. Segments are separated by zero values in the cell slice.
// Returns an error if the source has fewer than two points or if no segments
// with enough points remain after splitting.
func NewCells(src TrackSource, resolution int) (*Cells, error) {
	pts := []Point{}
	for p := range src.All() {
		pts = append(pts, p)
	}
	pointSegs, err := checkAndSplit(pts, src.Metadata().TrackType == TrackTypeRecorded)
	if err != nil {
		return nil, err
	}

	if len(pointSegs) == 0 {
		return nil, fmt.Errorf("no segments with enough points")
	}

	track := Cells{
		cells:  []h3.Cell{},
		nZeros: len(pointSegs) - 1,
		res:    resolution,
	}

	for i, pts := range pointSegs {
		track.cells = append(track.cells, interpolatePoints(pts, resolution)...)
		if i < track.nZeros {
			track.cells = append(track.cells, 0)
		}
	}

	return &track, nil
}

type cellsGob struct {
	Meta   Metadata
	Points []Point
	Cells  []h3.Cell
	NZeros int
	Res    int
}

// CellsToBytes serializes Cells to a gob-encoded byte slice.
func (c *Cells) CellsToBytes() ([]byte, error) {
	m := cellsGob{
		Cells:  c.cells,
		NZeros: c.nZeros,
		Res:    c.res,
	}

	buf := bytes.Buffer{}
	err := gob.NewEncoder(&buf).Encode(m)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// CellsFromBytes deserializes Cells from a gob-encoded byte slice.
func CellsFromBytes(data []byte) (*Cells, error) {
	var m cellsGob
	buf := bytes.NewBuffer(data)
	err := gob.NewDecoder(buf).Decode(&m)
	if err != nil {
		return nil, err
	}

	return &Cells{
		cells:  m.Cells,
		nZeros: m.NZeros,
		res:    m.Res,
	}, nil
}
