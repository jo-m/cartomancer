package grib2

import (
	"fmt"
	"io"
	"math"
)

const (
	// gridCellSize is the spatial bucket size used in the spatial index,
	// in degrees.  At 0.01° ≈ 1.1 km it matches the ICON-CH1 ~1 km resolution,
	// so most cells contain exactly one grid point.
	gridCellSize = float32(0.01)

	// gridSearchRadius is the number of cell rings to search when looking for
	// the nearest grid point.  2 guarantees correctness even near cell boundaries.
	gridSearchRadius = 2

	// ICON-CH1 horizontal-constants parameter codes used to identify the
	// center-latitude (CLAT) and center-longitude (CLON) messages.
	clonCategory  = uint8(191)
	clonParameter = uint8(2)
	clatCategory  = uint8(191)
	clatParameter = uint8(1)
)

// Grid holds the spatial coordinates of the ICON unstructured triangular grid
// and a bucket-based spatial index for fast nearest-neighbour lookups.
type Grid struct {
	// Lats contains the geodetic latitude (degrees N) of each grid cell centre.
	// Indices correspond to GRIB2 grid-point order.
	Lats []float32
	// Lons contains the geodetic longitude (degrees E) of each grid cell centre.
	Lons []float32

	// Spatial index: maps a bucket key (derived from lat/lon rounded to
	// gridCellSize) to the list of grid-point indices whose centre falls in
	// that bucket.
	index map[int64][]int32
}

// ParseGrid reads the horizontal-constants GRIB2 file for the ICON-CH1 model
// and returns a Grid populated with the centre lat/lon of every grid cell.
//
// The horizontal-constants file contains five GRIB2 messages; ParseGrid
// extracts the CLAT (cat=191, param=1) and CLON (cat=191, param=2) messages
// and builds a spatial index suitable for NearestIndex lookups.
func ParseGrid(r io.Reader) (*Grid, error) {
	msgs, err := Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parsing grid GRIB2: %w", err)
	}

	var lats, lons []float32
	for _, m := range msgs {
		switch {
		case m.Category == clatCategory && m.Parameter == clatParameter:
			lats = m.Values
		case m.Category == clonCategory && m.Parameter == clonParameter:
			lons = m.Values
		}
	}

	if lats == nil {
		return nil, fmt.Errorf("CLAT message (cat=%d, param=%d) not found in horizontal constants", clatCategory, clatParameter)
	}
	if lons == nil {
		return nil, fmt.Errorf("CLON message (cat=%d, param=%d) not found in horizontal constants", clonCategory, clonParameter)
	}
	if len(lats) != len(lons) {
		return nil, fmt.Errorf("CLAT (%d) and CLON (%d) point counts differ", len(lats), len(lons))
	}

	g := &Grid{Lats: lats, Lons: lons}
	g.buildIndex()
	return g, nil
}

// buildIndex populates the spatial bucket index from the grid coordinates.
func (g *Grid) buildIndex() {
	g.index = make(map[int64][]int32, len(g.Lats))
	for i, lat := range g.Lats {
		key := bucketKey(lat, g.Lons[i])
		g.index[key] = append(g.index[key], int32(i))
	}
}

// NearestIndex returns the index of the grid point whose centre is closest to
// (lat, lon) in degrees using the squared Euclidean distance in lat/lon space.
// It returns -1 if the grid is empty or if no point is found within the search
// radius (which indicates a query far outside the model domain).
func (g *Grid) NearestIndex(lat, lon float64) int {
	ilat := int32(math.Round(float64(float32(lat)) / float64(gridCellSize)))
	ilon := int32(math.Round(float64(float32(lon)) / float64(gridCellSize)))

	bestIdx := -1
	bestDist := float32(math.MaxFloat32)

	for dlat := int32(-gridSearchRadius); dlat <= gridSearchRadius; dlat++ {
		for dlon := int32(-gridSearchRadius); dlon <= gridSearchRadius; dlon++ {
			key := int64(ilat+dlat)*1_000_000 + int64(ilon+dlon)
			for _, idx := range g.index[key] {
				dlat2 := g.Lats[idx] - float32(lat)
				dlon2 := g.Lons[idx] - float32(lon)
				d := dlat2*dlat2 + dlon2*dlon2
				if d < bestDist {
					bestDist = d
					bestIdx = int(idx)
				}
			}
		}
	}
	return bestIdx
}

// ValueAt returns the decoded field value from msg at the grid point nearest to
// (lat, lon).  It returns NaN if no grid point is found (e.g. query outside the
// model domain).
func (g *Grid) ValueAt(msg *Message, lat, lon float64) float32 {
	idx := g.NearestIndex(lat, lon)
	if idx < 0 || idx >= len(msg.Values) {
		return float32(math.NaN())
	}
	return msg.Values[idx]
}

// bucketKey computes the spatial bucket key for a grid point at (lat, lon).
func bucketKey(lat, lon float32) int64 {
	ilat := int32(math.Round(float64(lat) / float64(gridCellSize)))
	ilon := int32(math.Round(float64(lon) / float64(gridCellSize)))
	return int64(ilat)*1_000_000 + int64(ilon)
}
