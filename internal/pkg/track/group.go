package track

import (
	"errors"

	"github.com/uber/h3-go/v4"
	"gonum.org/v1/gonum/stat/combin"
)

// pair is an unordered pair of track indices, normalized so that a <= b.
type pair struct {
	a, b int
}

// newPair creates a pair with indices in canonical order (a <= b).
func newPair(a, b int) pair {
	if a > b {
		a, b = b, a
	}
	return pair{a: a, b: b}
}

// GroupResult holds the output of Group: a list of groups (each a set of track indices)
// and a list of track indices that did not match any group.
type GroupResult struct {
	Groups     []map[int]struct{}
	NotMatched []int
}

// Group clusters tracks by shared H3 directed edges into groups of similar routes.
// For each pair of tracks, it counts how many directed edges they share and computes
// the ratio of shared edges to total edges for each track. If the minimum of the two
// ratios exceeds matchMinRatio, the tracks are placed in the same group. Tracks that
// do not match any other track are returned in NotMatched. All tracks must use the
// same H3 resolution.
func Group(tracks []*Cells, matchMinRatio float64) (*GroupResult, error) {
	if len(tracks) == 0 {
		return nil, errors.New("no cells to group")
	}

	tracksPerEdge := map[h3.DirectedEdge]map[int]struct{}{}

	res := tracks[0].res
	for i, track := range tracks {
		if track.res != res {
			return nil, errors.New("tracks have different resolutions")
		}

		for j, c1 := range track.cells[1:] {
			c0 := track.cells[j]

			if c0 == 0 || c1 == 0 {
				continue
			}

			edge, err := c0.DirectedEdge(c1)
			if err != nil {
				panic(err)
			}

			if tracksPerEdge[edge] == nil {
				tracksPerEdge[edge] = map[int]struct{}{}
			}
			tracksPerEdge[edge][i] = struct{}{}
		}
	}

	pairsCommon := map[pair]int{}

	for _, tracks := range tracksPerEdge {
		if len(tracks) < 2 {
			continue
		}

		tracksList := make([]int, 0, len(tracks))
		for i := range tracks {
			tracksList = append(tracksList, i)
		}

		for _, comb := range combin.Combinations(len(tracksList), 2) {
			p := newPair(tracksList[comb[0]], tracksList[comb[1]])
			pairsCommon[p]++
		}
	}

	// List of groups.
	groups := []map[int]struct{}{}
	// For each track, the group it belongs to.
	groupIndex := map[int]int{}
	for p, common := range pairsCommon {
		ratioA := float64(common) / float64(tracks[p.a].NEdges())
		ratioB := float64(common) / float64(tracks[p.b].NEdges())

		minRatio := ratioA
		if ratioB < minRatio {
			minRatio = ratioB
		}

		if minRatio < matchMinRatio {
			continue
		}

		groupA, okA := groupIndex[p.a]
		groupB, okB := groupIndex[p.b]
		switch {
		case okA && okB && groupA == groupB:
			// Already in the same group.
		case okA && okB:
			// Merge group B into group A.
			for member := range groups[groupB] {
				groups[groupA][member] = struct{}{}
				groupIndex[member] = groupA
			}
			groups[groupB] = nil
		case okA:
			groups[groupA][p.b] = struct{}{}
			groupIndex[p.b] = groupA
		case okB:
			groups[groupB][p.a] = struct{}{}
			groupIndex[p.a] = groupB
		default:
			group := map[int]struct{}{
				p.a: {},
				p.b: {},
			}
			groups = append(groups, group)
			groupIndex[p.a] = len(groups) - 1
			groupIndex[p.b] = len(groups) - 1
		}
	}

	// Filter out nil groups left behind by merges.
	compacted := make([]map[int]struct{}, 0, len(groups))
	for _, g := range groups {
		if g != nil {
			compacted = append(compacted, g)
		}
	}

	ret := &GroupResult{
		Groups:     compacted,
		NotMatched: []int{},
	}

	for i := range tracks {
		if _, ok := groupIndex[i]; !ok {
			ret.NotMatched = append(ret.NotMatched, i)
		}
	}

	return ret, nil
}
