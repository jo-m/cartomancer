// Package segment extracts shared road/way segments from GPS tracks.
//
// The algorithm is structured as a pipeline of independent phases, each
// exposed as a public function so that callers can inspect intermediate
// results (e.g. for visualization in segviz):
//
//  1. [DetectJunctions] — walks all tracks incrementally, building a directed
//     edge map, and marks junctions where the track-ID set changes between
//     consecutive edges.
//  2. [RefineJunctions] — clusters nearby raw junctions and snaps each
//     cluster to actual GPS track points at high-resolution H3 cells,
//     selecting the location with the highest track overlap.
//  3. [SliceAtJunctions] — walks each track again, splitting its cell
//     sequence at every junction cell. Segments with identical cell
//     sequences (direction-sensitive) are deduplicated.
//  4. [FilterSegments] — removes segments shorter than [MinSegmentDistanceM]
//     or shared by fewer than minTracks.
//  5. [AttachPolylines] — loads GPS points from a representative member
//     track on-demand to produce map-ready polylines that follow the actual
//     road geometry instead of H3 cell centers.
//
// [Extract] is a convenience wrapper that runs phases 1–4 in order and
// returns all intermediate results in an [ExtractResult].
//
// # Integration
//
// Segment extraction is triggered as a background job (see job.go) whenever
// tracks are uploaded. The job loads all tracks, runs [Extract], attaches
// polylines, and replaces the stored segments in a single transaction.
package segment

import (
	"encoding/json"
	"fmt"

	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

const (
	// DefaultResolution is the default H3 resolution for segment extraction.
	// See https://h3geo.org/docs/core-library/restable/#edge-lengths.
	DefaultResolution = 11

	// MinTrackCount is the minimum number of tracks that must traverse a
	// segment for it to be kept.
	MinTrackCount = 1

	// MinSegmentDistanceM is the minimum distance in meters for a segment
	// to be kept. Shorter segments are discarded as noise.
	MinSegmentDistanceM = 50.0

	// clusterDistanceM is the maximum distance in meters between two raw
	// junctions for them to be merged into the same cluster during
	// refinement.
	clusterDistanceM = 100.0

	// clusterGridK is the k-ring radius used when searching for neighboring
	// junctions during union-find clustering. At resolution 10, k=2 covers
	// approximately 132 m from the center cell.
	clusterGridK = 2

	// Finer H3 resolution for snapping junctions to actual GPS track points.
	// See https://h3geo.org/docs/core-library/restable/#edge-lengths.
	snapResolution = 12

	// snapNeighborK is the k-ring radius at segment resolution used to
	// define the search area around a junction cluster when collecting
	// nearby GPS points for snapping. At resolution 10, k=1 covers the
	// immediate hex neighbors (~66 m).
	snapNeighborK = 1
)

// trackUUID is a typed wrapper for track UUID strings used internally to
// distinguish track identifiers from arbitrary strings.
type trackUUID string

// TrackCells pairs a track UUID with its H3 cell representation.
type TrackCells struct {
	UUID  string
	Cells *track.Cells
}

// trackID returns the track's UUID as a typed [trackUUID].
func (tc TrackCells) trackID() trackUUID { return trackUUID(tc.UUID) }

// Junction is a point where tracks diverge or converge.
type Junction struct {
	H3Cell h3.Cell
	Lat    float64
	Lon    float64
}

// Segment is a maximal path between two junctions shared by one or more tracks.
type Segment struct {
	StartJunction Junction
	EndJunction   Junction
	// Cells is the ordered sequence of H3 cells forming this segment.
	Cells []h3.Cell
	// Polyline is the representative GPS coordinates for this segment,
	// extracted from one of the member tracks. Populated by [AttachPolylines].
	Polyline   [][2]float64
	TrackUUIDs []string
	DistanceM  float64
}

// ExtractResult holds the output of [Extract], including final and
// intermediate results from each pipeline phase.
type ExtractResult struct {
	// RawJunctions is the junction set from phase 1 (before refinement).
	RawJunctions map[h3.Cell]struct{}
	// RefinedJunctions is the junction set from phase 2 (after refinement).
	// Nil if no [RawPointLoader] was provided.
	RefinedJunctions map[h3.Cell]struct{}
	// RawSegments contains all deduplicated segments before filtering.
	RawSegments []Segment
	// Segments contains the final filtered segments.
	Segments []Segment
}

// PolylineJSON returns the segment's polyline as a JSON array of [lat,lon] pairs.
func (s *Segment) PolylineJSON() (string, error) {
	b, err := json.Marshal(s.Polyline)
	if err != nil {
		return "", fmt.Errorf("marshalling polyline: %w", err)
	}
	return string(b), nil
}

// RawPointLoader loads all GPS points for a track given its UUID.
// Unlike [PointLoader], it returns every recorded point without cell
// indexing, preserving the original GPS density at direction changes.
type RawPointLoader func(uuid string) ([]track.Point, error)

// dirEdge is a directed edge between two H3 cells.
type dirEdge = [2]h3.Cell

// cellLatLng returns the center coordinates of an H3 cell. Panics on invalid
// cells, which is safe because all cells come from validated H3 operations.
func cellLatLng(c h3.Cell) h3.LatLng {
	ll, err := c.LatLng()
	if err != nil {
		panic(err)
	}
	return ll
}
