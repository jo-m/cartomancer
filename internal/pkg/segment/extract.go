package segment

import (
	"fmt"
	"slices"
	"strings"

	"github.com/uber/h3-go/v4"
)

// SliceAtJunctions walks each track's cell sequence and splits it at every
// junction cell. The junction cell is included in both adjacent segments to
// avoid gaps. Track UUIDs for each segment are determined by intersecting
// the deduplicated UUID sets of all its directed edges.
//
// Segments with identical cell sequences (direction-sensitive) are
// deduplicated, merging their track UUID sets.
func SliceAtJunctions(tracks []TrackCells, junctions map[h3.Cell]struct{}, ei *EdgeIndex) []Segment {
	type dedupEntry struct {
		cells []h3.Cell
		uuids map[trackUUID]struct{}
	}
	dedup := make(map[string]*dedupEntry)

	flushSeg := func(segCells []h3.Cell) {
		if len(segCells) < 2 {
			return
		}

		// Determine track UUIDs as the intersection of all edges'
		// deduplicated UUID sets.
		var uuids map[trackUUID]struct{}
		for k := 1; k < len(segCells); k++ {
			edgeList := ei.edges[dirEdge{segCells[k-1], segCells[k]}]
			edgeUUIDs := make(map[trackUUID]struct{}, len(edgeList))
			for _, uuid := range edgeList {
				edgeUUIDs[uuid] = struct{}{}
			}
			if uuids == nil {
				uuids = edgeUUIDs
			} else {
				for uuid := range uuids {
					if _, ok := edgeUUIDs[uuid]; !ok {
						delete(uuids, uuid)
					}
				}
			}
		}
		if len(uuids) == 0 {
			return
		}

		key := cellKey(segCells)
		if entry, ok := dedup[key]; ok {
			for uuid := range uuids {
				entry.uuids[uuid] = struct{}{}
			}
		} else {
			dedup[key] = &dedupEntry{
				cells: append([]h3.Cell{}, segCells...),
				uuids: uuids,
			}
		}
	}

	for _, tc := range tracks {
		allCells := tc.Cells.AllCells()
		var segCells []h3.Cell

		for _, c := range allCells {
			if c == 0 {
				flushSeg(segCells)
				segCells = nil
				continue
			}

			if len(segCells) > 0 && segCells[len(segCells)-1] == c {
				continue
			}

			segCells = append(segCells, c)

			if _, isJunction := junctions[c]; isJunction && len(segCells) >= 2 {
				flushSeg(segCells)
				segCells = []h3.Cell{c}
			}
		}
		flushSeg(segCells)
	}

	// Build Segment structs from deduplicated entries.
	result := make([]Segment, 0, len(dedup))
	for _, entry := range dedup {
		uuids := make([]string, 0, len(entry.uuids))
		for uuid := range entry.uuids {
			uuids = append(uuids, string(uuid))
		}
		slices.Sort(uuids)

		startCell := entry.cells[0]
		endCell := entry.cells[len(entry.cells)-1]
		startLL := cellLatLng(startCell)
		endLL := cellLatLng(endCell)

		distM := 0.0
		for k := 1; k < len(entry.cells); k++ {
			distM += h3.GreatCircleDistanceM(cellLatLng(entry.cells[k-1]), cellLatLng(entry.cells[k]))
		}

		result = append(result, Segment{
			StartJunction: Junction{H3Cell: startCell, Lat: startLL.Lat, Lon: startLL.Lng},
			EndJunction:   Junction{H3Cell: endCell, Lat: endLL.Lat, Lon: endLL.Lng},
			Cells:         entry.cells,
			TrackUUIDs:    uuids,
			DistanceM:     distM,
		})
	}

	return result
}

// FilterSegments removes segments shorter than minDistM or shared by fewer
// than minTracks.
func FilterSegments(segments []Segment, minTracks int, minDistM float64) []Segment {
	var result []Segment
	for _, seg := range segments {
		if len(seg.TrackUUIDs) < minTracks {
			continue
		}
		if seg.DistanceM < minDistM {
			continue
		}
		result = append(result, seg)
	}
	return result
}

// Extract runs the full extraction pipeline: junction detection, optional
// refinement, slicing, and filtering. It is a convenience wrapper around the
// individual phase functions for callers that do not need intermediate results.
//
// The returned [ExtractResult] contains segments with Cells populated but
// Polyline empty, plus the raw and refined junction cell sets.
// Call [AttachPolylines] afterwards to fill in polylines from GPS points.
func Extract(tracks []TrackCells, minTracks int, loader RawPointLoader) (*ExtractResult, error) {
	if len(tracks) == 0 {
		return &ExtractResult{}, nil
	}

	rawJunctions, edgeIndex := DetectJunctions(tracks)

	var refinedJunctions map[h3.Cell]struct{}
	junctions := rawJunctions
	if loader != nil {
		resolution := tracks[0].Cells.Resolution()
		refined, err := RefineJunctions(rawJunctions, tracks, loader, resolution)
		if err != nil {
			return nil, fmt.Errorf("refining junctions: %w", err)
		}
		refinedJunctions = refined
		junctions = refined
	}

	rawSegments := SliceAtJunctions(tracks, junctions, edgeIndex)
	segments := FilterSegments(rawSegments, minTracks, MinSegmentDistanceM)

	return &ExtractResult{
		RawJunctions:     rawJunctions,
		RefinedJunctions: refinedJunctions,
		RawSegments:      rawSegments,
		Segments:         segments,
	}, nil
}

// cellKey returns a deduplication key for an ordered cell sequence.
func cellKey(cells []h3.Cell) string {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = c.String()
	}
	return strings.Join(parts, ",")
}
