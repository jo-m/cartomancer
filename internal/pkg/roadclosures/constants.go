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
