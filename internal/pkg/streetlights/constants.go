// Package streetlights provides shared types and helpers for ingesting street
// light features from multiple upstream sources. Source-specific clients and
// jobs live in subpackages (e.g. ktzh, stadtzh).
package streetlights

// CellResolution is the H3 resolution used to index streetlight points for
// spatial lookups. It is finer than the roadclosures package's resolution 7,
// because lamps are dense point features: a single resolution 7 cell can
// contain well over a thousand lamps in the city of Zurich.
const CellResolution = 9
