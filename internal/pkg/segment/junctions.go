package segment

import (
	"math"
	"slices"

	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/detour/internal/pkg/track"
)

// EdgeIndex holds the directed edge map built during junction detection.
// It is opaque to callers but must be passed to [SliceAtJunctions] so that
// track UUIDs can be resolved per segment.
type EdgeIndex struct {
	edges map[dirEdge][]trackUUID
}

// DetectJunctions walks all tracks, incrementally building a directed edge
// map, and detects junction cells where the track-ID list changes between
// consecutive edges. A junction indicates a point where a track joins or
// leaves others.
//
// Each track is processed in both forward and reverse direction. The reverse
// pass uses a separate edge map but writes into the same junction set. This
// catches junction candidates at convergence points that the forward pass
// misses — for example when two tracks run parallel within one H3 cell
// width before merging, causing the exact join cell to shift due to cell
// boundary quantization.
//
// Returns the set of junction cells and an [EdgeIndex] that records which
// track UUIDs traverse each directed edge (forward only, used by
// [SliceAtJunctions]).
func DetectJunctions(tracks []TrackCells) (map[h3.Cell]struct{}, *EdgeIndex) {
	fwdEdgeMap := make(map[dirEdge][]trackUUID)
	junctions := make(map[h3.Cell]struct{})

	// Forward pass: builds the authoritative edge map and detects junctions.
	detectPass(tracks, fwdEdgeMap, junctions, false)

	// Reverse pass: walks each track in reverse with a separate edge map,
	// surfacing additional junction candidates that the forward pass missed.
	revEdgeMap := make(map[dirEdge][]trackUUID)
	detectPass(tracks, revEdgeMap, junctions, true)

	// Cell-occupancy pass: pre-computes which tracks occupy each cell,
	// then walks every track marking junctions where the occupying track
	// set changes between consecutive cells. Unlike the edge-based passes
	// this is order- and direction-independent, catching cases where
	// parallel tracks converge or diverge within a single H3 cell width.
	detectCellOccupancyJunctions(tracks, junctions)

	// Self-crossing pass: detects cells that a single track visits more
	// than once. The edge-based and cell-occupancy passes cannot catch
	// these because they only compare across tracks, not within a track.
	detectSelfCrossings(tracks, junctions)

	return junctions, &EdgeIndex{edges: fwdEdgeMap}
}

// detectPass walks each track's cell sequence (optionally reversed),
// building directed edges in edgeMap and recording junction cells where
// the track-ID list changes between consecutive edges.
func detectPass(tracks []TrackCells, edgeMap map[dirEdge][]trackUUID, junctions map[h3.Cell]struct{}, reverse bool) {
	for _, tc := range tracks {
		allCells := tc.Cells.AllCells()
		if reverse {
			allCells = reversedCells(allCells)
		}

		var prevEdgeList []trackUUID
		var prevCell h3.Cell
		edgeStarted := false

		for _, c := range allCells {
			if c == 0 {
				// Track discontinuity. Check if the last edge had other
				// tracks (current track is "leaving" them).
				if edgeStarted && len(prevEdgeList) > 0 {
					junctions[prevCell] = struct{}{}
				}
				prevCell = 0
				prevEdgeList = nil
				edgeStarted = false
				continue
			}

			if prevCell == 0 {
				prevCell = c
				continue
			}
			if prevCell == c {
				continue
			}

			e := dirEdge{prevCell, c}
			currentList := edgeMap[e]

			if !edgeStarted {
				// First edge in this continuous segment. If other tracks
				// already use this edge, we are "joining" them here.
				if len(currentList) > 0 {
					junctions[prevCell] = struct{}{}
				}
				edgeStarted = true
			} else if !slices.Equal(prevEdgeList, currentList) {
				junctions[prevCell] = struct{}{}
			}

			prevEdgeList = currentList
			edgeMap[e] = append(edgeMap[e], tc.trackID())
			prevCell = c
		}

		// End of track: if the last edge had other tracks, the current
		// track is "leaving" them at the last cell.
		if edgeStarted && len(prevEdgeList) > 0 {
			junctions[prevCell] = struct{}{}
		}
	}
}

// reversedCells returns a reversed copy of a cell slice, preserving zero
// separators at discontinuity boundaries.
func reversedCells(cells []h3.Cell) []h3.Cell {
	rev := make([]h3.Cell, len(cells))
	for i, c := range cells {
		rev[len(cells)-1-i] = c
	}
	return rev
}

// detectSelfCrossings walks each track's cell sequence and marks any cell
// that the track visits more than once as a junction. This catches loops
// and figure-eight patterns that the edge-based and cell-occupancy passes
// miss because those only compare across tracks.
func detectSelfCrossings(tracks []TrackCells, junctions map[h3.Cell]struct{}) {
	for _, tc := range tracks {
		visited := make(map[h3.Cell]struct{})
		for _, c := range tc.Cells.AllCells() {
			if c == 0 {
				visited = make(map[h3.Cell]struct{})
				continue
			}
			if _, seen := visited[c]; seen {
				junctions[c] = struct{}{}
			}
			visited[c] = struct{}{}
		}
	}
}

// detectCellOccupancyJunctions builds a cell→trackUUID set from all tracks,
// then walks each track's cell sequence and marks a junction wherever the
// set of occupying tracks changes between consecutive cells. This is both
// order- and direction-independent, complementing the edge-based passes.
func detectCellOccupancyJunctions(tracks []TrackCells, junctions map[h3.Cell]struct{}) {
	cellTracks := make(map[h3.Cell]map[trackUUID]struct{})
	for _, tc := range tracks {
		for _, c := range tc.Cells.AllCells() {
			if c == 0 {
				continue
			}
			m := cellTracks[c]
			if m == nil {
				m = make(map[trackUUID]struct{})
				cellTracks[c] = m
			}
			m[tc.trackID()] = struct{}{}
		}
	}

	for _, tc := range tracks {
		var prevCell h3.Cell
		var prevSet map[trackUUID]struct{}

		for _, c := range tc.Cells.AllCells() {
			if c == 0 {
				prevCell = 0
				prevSet = nil
				continue
			}
			if prevCell == 0 {
				prevCell = c
				prevSet = cellTracks[c]
				continue
			}
			if prevCell == c {
				continue
			}

			curSet := cellTracks[c]
			if !trackSetsEqual(prevSet, curSet) {
				// Mark the cell on the "shared" side: for divergence
				// (more tracks on prevCell) mark prevCell, for
				// convergence (more tracks on c) mark c. When the sets
				// are equal-sized but differ in composition, mark both.
				if len(prevSet) >= len(curSet) {
					junctions[prevCell] = struct{}{}
				}
				if len(curSet) >= len(prevSet) {
					junctions[c] = struct{}{}
				}
			}

			prevCell = c
			prevSet = curSet
		}
	}
}

// trackSetsEqual reports whether two track UUID sets contain the same
// elements.
func trackSetsEqual(a, b map[trackUUID]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// RefineJunctions clusters nearby raw junctions and snaps each cluster to
// the raw GPS track point at the location with the highest track overlap.
//
//  1. Union-find clustering: junctions within [clusterDistanceM] are merged.
//  2. For each cluster, raw GPS points from nearby tracks are filtered to
//     the cluster's segment-resolution neighborhood, then indexed into
//     high-res H3 cells (segment resolution + [snapResOffset]), recording
//     which track UUIDs occupy each cell.
//  3. The high-res cell with the most distinct track UUIDs is picked (ties
//     broken by proximity to the cluster centroid). Its GPS point is mapped
//     back to the segment resolution to produce the refined junction.
func RefineJunctions(
	junctions map[h3.Cell]struct{},
	tracks []TrackCells,
	loader RawPointLoader,
	resolution int,
) (map[h3.Cell]struct{}, error) {
	if len(junctions) == 0 {
		return junctions, nil
	}

	// Step 1: cluster nearby junctions using union-find.
	uf := make(map[h3.Cell]h3.Cell, len(junctions))
	for j := range junctions {
		uf[j] = j
	}

	var find func(h3.Cell) h3.Cell
	find = func(c h3.Cell) h3.Cell {
		if uf[c] != c {
			uf[c] = find(uf[c])
		}
		return uf[c]
	}
	union := func(a, b h3.Cell) {
		ra, rb := find(a), find(b)
		if ra != rb {
			uf[ra] = rb
		}
	}

	for j := range junctions {
		jLL := cellLatLng(j)
		neighbors, err := h3.GridDisk(j, clusterGridK)
		if err != nil {
			continue
		}
		for _, nc := range neighbors {
			if nc == 0 || nc == j {
				continue
			}
			if _, ok := junctions[nc]; !ok {
				continue
			}
			ncLL := cellLatLng(nc)
			if h3.GreatCircleDistanceM(jLL, ncLL) <= clusterDistanceM {
				union(j, nc)
			}
		}
	}

	// Step 2: group junctions by cluster root, compute cluster centroids.
	type cluster struct {
		members []h3.Cell
		sumLat  float64
		sumLon  float64
	}
	clusters := make(map[h3.Cell]*cluster)
	for j := range junctions {
		root := find(j)
		ll := cellLatLng(j)
		cl := clusters[root]
		if cl == nil {
			cl = &cluster{}
			clusters[root] = cl
		}
		cl.members = append(cl.members, j)
		cl.sumLat += ll.Lat
		cl.sumLon += ll.Lng
	}

	// Build cell -> track UUID set at segment resolution for neighbor lookup.
	cellTracks := make(map[h3.Cell]map[trackUUID]struct{})
	for _, tc := range tracks {
		for _, c := range tc.Cells.AllCells() {
			if c == 0 {
				continue
			}
			m := cellTracks[c]
			if m == nil {
				m = make(map[trackUUID]struct{})
				cellTracks[c] = m
			}
			m[tc.trackID()] = struct{}{}
		}
	}

	// Step 3: pre-index raw GPS points from tracks near junctions by their
	// segment-resolution cell. This lets each cluster look up only the
	// points in its neighborhood without scanning entire tracks.
	type taggedPoint struct {
		uuid trackUUID
		pt   track.Point
	}

	allJunctionUUIDs := make(map[trackUUID]struct{})
	for j := range junctions {
		disk, err := h3.GridDisk(j, snapNeighborK)
		if err != nil {
			continue
		}
		for _, nc := range disk {
			if nc == 0 {
				continue
			}
			for uuid := range cellTracks[nc] {
				allJunctionUUIDs[uuid] = struct{}{}
			}
		}
	}

	segCellPoints := make(map[h3.Cell][]taggedPoint)
	for uuid := range allJunctionUUIDs {
		pts, loadErr := loader(string(uuid))
		if loadErr != nil || len(pts) == 0 {
			continue
		}
		for _, pt := range pts {
			segCell := pt.Cell(resolution)
			segCellPoints[segCell] = append(segCellPoints[segCell], taggedPoint{uuid: uuid, pt: pt})
		}
	}

	// Step 4: for each cluster, collect nearby raw points via the
	// segment-res index, compute high-res cells, and pick the one with
	// the most track overlap.
	refined := make(map[h3.Cell]struct{}, len(clusters))

	for _, cl := range clusters {
		n := float64(len(cl.members))
		centLat := cl.sumLat / n
		centLon := cl.sumLon / n

		type hiResInfo struct {
			uuids map[trackUUID]struct{}
			pt    track.Point
		}
		hiResCells := make(map[h3.Cell]*hiResInfo)

		for _, m := range cl.members {
			disk, err := h3.GridDisk(m, snapNeighborK)
			if err != nil {
				continue
			}
			for _, nc := range disk {
				if nc == 0 {
					continue
				}
				for _, tp := range segCellPoints[nc] {
					hiCell := tp.pt.Cell(snapResolution)
					info := hiResCells[hiCell]
					if info == nil {
						info = &hiResInfo{
							uuids: make(map[trackUUID]struct{}),
							pt:    tp.pt,
						}
						hiResCells[hiCell] = info
					}
					info.uuids[tp.uuid] = struct{}{}
				}
			}
		}

		bestCount := 0
		bestDistSq := math.MaxFloat64
		var bestPt track.Point

		for _, info := range hiResCells {
			cnt := len(info.uuids)
			dLat := info.pt.Lat - centLat
			dLon := info.pt.Lon - centLon
			distSq := dLat*dLat + dLon*dLon

			if cnt > bestCount || (cnt == bestCount && distSq < bestDistSq) {
				bestCount = cnt
				bestDistSq = distSq
				bestPt = info.pt
			}
		}

		if bestCount == 0 {
			centroidPt := track.Point{Lat: centLat, Lon: centLon}
			refined[centroidPt.Cell(resolution)] = struct{}{}
		} else {
			refined[bestPt.Cell(resolution)] = struct{}{}
		}
	}

	return refined, nil
}
