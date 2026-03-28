package segment

import (
	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/detour/internal/pkg/track"
)

// PointLoader loads GPS points for a track given its UUID at the specified
// H3 resolution. Returns a map from cell to the first GPS point that falls
// in that cell. Used by [AttachPolylines] to load points on-demand instead
// of preloading all tracks.
type PointLoader func(uuid string, resolution int) (map[h3.Cell]track.Point, error)

// AttachPolylines fills in the Polyline and DistanceM fields of each segment
// by loading GPS points from representative member tracks on-demand.
// For each segment, the track with the best cell coverage is selected and its
// GPS coordinates replace the H3 cell centers, producing polylines that
// follow actual road geometry.
func AttachPolylines(segments []Segment, loader PointLoader, resolution int) error {
	cache := make(map[trackUUID]map[h3.Cell]track.Point)

	for i := range segments {
		seg := &segments[i]

		var bestUUID trackUUID
		bestCount := 0

		for _, uuid := range seg.TrackUUIDs {
			cellMap, err := getCachedPoints(trackUUID(uuid), resolution, loader, cache)
			if err != nil {
				continue
			}
			count := 0
			for _, c := range seg.Cells {
				if _, ok := cellMap[c]; ok {
					count++
				}
			}
			if count > bestCount {
				bestCount = count
				bestUUID = trackUUID(uuid)
			}
		}

		if bestUUID == "" || bestCount == 0 {
			seg.Polyline, seg.DistanceM = fallbackPolyline(seg.Cells)
			continue
		}

		cellMap := cache[bestUUID]
		polyline := make([][2]float64, 0, len(seg.Cells))
		for _, c := range seg.Cells {
			if p, ok := cellMap[c]; ok {
				polyline = append(polyline, [2]float64{p.Lat, p.Lon})
			}
		}

		if len(polyline) < 2 {
			seg.Polyline, seg.DistanceM = fallbackPolyline(seg.Cells)
			continue
		}

		distM := 0.0
		for j := 1; j < len(polyline); j++ {
			p0 := track.Point{Lat: polyline[j-1][0], Lon: polyline[j-1][1]}
			p1 := track.Point{Lat: polyline[j][0], Lon: polyline[j][1]}
			distM += p0.MetersTo(&p1)
		}

		seg.Polyline = polyline
		seg.DistanceM = distM
	}

	return nil
}

// getCachedPoints returns the cell-to-point map for a track, using the cache
// or loading via the loader if not cached.
func getCachedPoints(uuid trackUUID, resolution int, loader PointLoader, cache map[trackUUID]map[h3.Cell]track.Point) (map[h3.Cell]track.Point, error) {
	if m, ok := cache[uuid]; ok {
		return m, nil
	}
	m, err := loader(string(uuid), resolution)
	if err != nil {
		return nil, err
	}
	cache[uuid] = m
	return m, nil
}

// fallbackPolyline generates a polyline from H3 cell centers when no track
// points are available.
func fallbackPolyline(segCells []h3.Cell) ([][2]float64, float64) {
	polyline := make([][2]float64, len(segCells))
	distM := 0.0
	for i, c := range segCells {
		ll := cellLatLng(c)
		polyline[i] = [2]float64{ll.Lat, ll.Lng}
		if i > 0 {
			distM += h3.GreatCircleDistanceM(cellLatLng(segCells[i-1]), ll)
		}
	}
	return polyline, distM
}
