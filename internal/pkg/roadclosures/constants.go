// Package roadclosures provides shared types and helpers for ingesting road
// closure features from multiple upstream sources and intersecting them with
// tracks. Source-specific clients and jobs live in subpackages (e.g. astra,
// zh).
package roadclosures

// CellResolution is the H3 resolution used for coarse spatial indexing.
// Both the DB index (road_closure_cells_res7) and the API lookup use this.
// See https://h3geo.org/docs/core-library/restable/.
const CellResolution = 7

// FineResolution is the H3 resolution used for fine-grained intersection
// checks between a closure and the points of a track.
const FineResolution = 12

// ClosureType identifies the kind of road closure.
// The integer values are stored in the database.
type ClosureType int

const (
	// ClosedWay indicates a road or path that is physically closed.
	ClosedWay ClosureType = iota + 1
	// Detour indicates a detour route around a closed section.
	Detour
	// Obstruction indicates a construction site or other obstacle that
	// impedes traffic without fully closing the road or providing a
	// dedicated detour route.
	Obstruction
)

// String returns the canonical string representation of the closure type,
// used in API responses.
func (t ClosureType) String() string {
	switch t {
	case ClosedWay:
		return "closed_way"
	case Detour:
		return "detour"
	case Obstruction:
		return "obstruction"
	default:
		return "unknown"
	}
}
