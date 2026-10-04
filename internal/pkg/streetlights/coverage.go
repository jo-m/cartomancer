package streetlights

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// LitRadiusM is the maximum distance in meters between a point on a track and
// a streetlight for that point to count as lit.
const LitRadiusM = 20.0

// SampleStepM is the fixed cumulative-distance interval in meters at which
// tracks are sampled for the coverage computation.
const SampleStepM = 5.0

// searchBoxHalfWidthM is the half-width of the equirectangular pre-filter box
// that rejects sample/lamp pairs before the expensive great-circle distance.
// The 25 percent margin over [LitRadiusM] covers the approximation error of
// the local tangent-plane projection.
const searchBoxHalfWidthM = LitRadiusM * 1.25

// metersPerDegreeLat is the length of one degree of latitude in meters used
// by the equirectangular pre-filter.
const metersPerDegreeLat = 111320.0

// LitStretch is a contiguous lit section of a track between two cumulative
// distances in meters from the start of the track.
type LitStretch struct {
	StartDistanceM float64
	EndDistanceM   float64
}

// Attribution is the data source credit of a streetlight that lit part of a
// track.
type Attribution struct {
	Text string
	Href string
}

// CoverageResult describes how much of a track lies within [LitRadiusM] of a
// streetlight, and where.
type CoverageResult struct {
	// TotalDistanceM is the distance covered by the sampled path in meters.
	TotalDistanceM float64
	// LitDistanceM is the sum of all stretch lengths in meters.
	LitDistanceM float64
	// Stretches are the lit sections in ascending distance order.
	Stretches []LitStretch
	// Attributions credits the data sources that lit the track. It contains
	// only sources that actually lit at least one sample.
	Attributions []Attribution
}

// ComputeTrackCoverage samples the track's 5 m preview polyline at
// [SampleStepM] intervals and reports how much of the path lies within
// [LitRadiusM] of a streetlight, and where. The 5 m polyline is within 5 m of
// the recorded path and stretch boundaries are quantized by at most
// [SampleStepM] / 2, so the effective lit boundary is 20 m +/- 7.5 m worst
// case. A fully lit track reports LitDistanceM equal to TotalDistanceM.
//
// Tracks with fewer than two polyline points return a zero result with
// TotalDistanceM taken from the track metadata and empty, non-nil slices.
//
// Returns an error if the polyline cannot be decoded or the streetlight data
// cannot be loaded.
func ComputeTrackCoverage(ctx context.Context, d *db.DB, t db.Track) (CoverageResult, error) {
	res := CoverageResult{
		TotalDistanceM: t.TotalDistanceM,
		Stretches:      []LitStretch{},
		Attributions:   []Attribution{},
	}

	pts, err := track.DecodeVarint(t.PolylineDp5mVarint)
	if err != nil {
		return CoverageResult{}, fmt.Errorf("decode track polyline: %w", err)
	}
	samples := pts.InterpolateByDistance(SampleStepM)
	if len(samples) < 2 {
		return res, nil
	}
	res.TotalDistanceM = samples[len(samples)-1].DistanceM - samples[0].DistanceM

	cells, invalidSamples := sampleCells(samples)
	if invalidSamples > 0 {
		logg.Warn(ctx, "skipped track samples with invalid coordinates", "count", invalidSamples)
	}

	rows, err := d.GetStreetlightsByCells(ctx, expandCells(cells))
	if err != nil {
		return CoverageResult{}, fmt.Errorf("load streetlights along track: %w", err)
	}

	lamps, unusable := lampNeighbourhoods(rows)
	if unusable > 0 {
		logg.Warn(ctx, "skipped streetlights with unusable geometry", "count", unusable)
	}

	lit, attributions := litSamples(samples, cells, lamps)
	res.Stretches, res.LitDistanceM = buildStretches(samples, lit)
	res.Attributions = attributions
	return res, nil
}

// sampleCells returns the H3 cell at [CellResolution] of every sample, using
// the zero cell for samples whose coordinates could not be converted, along
// with the number of such invalid coordinates.
func sampleCells(samples []track.InterpolatedPoint) ([]h3.Cell, int) {
	cells := make([]h3.Cell, len(samples))
	invalid := 0
	for i, s := range samples {
		cell, err := h3.LatLngToCell(h3.LatLng{Lat: s.Lat, Lng: s.Lon}, CellResolution)
		if err != nil {
			invalid++
			continue
		}
		cells[i] = cell
	}
	return cells, invalid
}

// expandCells returns the sorted, deduplicated set of cells to load
// streetlights for: every valid sample cell plus its immediate neighbours.
// The k-ring is needed because a lamp within [LitRadiusM] of a sample can lie
// just across a cell boundary; the res-9 edge length is about 200 m. Around
// pentagons [h3.GridDisk] can fail or return nothing, in which case the cell
// itself is the safe fallback.
func expandCells(cells []h3.Cell) []int64 {
	seen := make(map[h3.Cell]struct{}, len(cells)*7)
	for _, c := range cells {
		if !c.IsValid() {
			continue
		}
		disk, err := h3.GridDisk(c, 1)
		if err != nil || len(disk) == 0 {
			disk = []h3.Cell{c}
		}
		for _, n := range disk {
			seen[n] = struct{}{}
		}
	}

	out := make([]int64, 0, len(seen))
	for c := range seen {
		out = append(out, int64(c))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// lamp is one streetlight as loaded from the database: a plain coordinate
// pair plus its source credit.
type lamp struct {
	lat, lon    float64
	attribution Attribution
}

// lampNeighbourhoods parses streetlight rows and groups them by their stored
// H3 cell. Rows whose geometry is missing, malformed, or not a finite point
// are skipped and counted.
func lampNeighbourhoods(rows []db.StreetlightByCell) (map[h3.Cell][]lamp, int) {
	byCell := make(map[h3.Cell][]lamp, len(rows))
	unusable := 0
	for _, row := range rows {
		geom, err := geojson.UnmarshalGeometry([]byte(row.Geometry))
		if err != nil {
			unusable++
			continue
		}
		p, ok := geom.Geometry().(orb.Point)
		if !ok || math.IsNaN(p.Lat()) || math.IsNaN(p.Lon()) || math.IsInf(p.Lat(), 0) || math.IsInf(p.Lon(), 0) {
			unusable++
			continue
		}
		cell := h3.Cell(row.Cell)
		byCell[cell] = append(byCell[cell], lamp{
			lat:         p.Lat(),
			lon:         p.Lon(),
			attribution: Attribution{Text: row.Attribution, Href: row.AttributionHref},
		})
	}
	return byCell, unusable
}

// litSamples decides for every sample whether a streetlight is within
// [LitRadiusM], scanning the lamps of the sample's cell and its neighbours.
// An equirectangular pre-filter rejects clearly distant pairs before the
// exact great-circle distance. It returns the per-sample lit flags and the
// deduplicated, sorted credits of the lamps nearest to lit samples.
func litSamples(samples []track.InterpolatedPoint, cells []h3.Cell, byCell map[h3.Cell][]lamp) ([]bool, []Attribution) {
	// ringCache memoizes the union of the lamps of one cell and its immediate
	// neighbours; consecutive samples usually share a cell.
	ringCache := make(map[h3.Cell][]lamp)
	ringLamps := func(cell h3.Cell) []lamp {
		if cached, ok := ringCache[cell]; ok {
			return cached
		}
		disk, err := h3.GridDisk(cell, 1)
		if err != nil || len(disk) == 0 {
			disk = []h3.Cell{cell}
		}
		var lamps []lamp
		for _, n := range disk {
			lamps = append(lamps, byCell[n]...)
		}
		ringCache[cell] = lamps
		return lamps
	}

	lit := make([]bool, len(samples))
	seen := make(map[Attribution]struct{})
	var attributions []Attribution
	for i, s := range samples {
		if !cells[i].IsValid() {
			continue
		}

		nearest := math.Inf(1)
		var nearestAttr Attribution
		for _, l := range ringLamps(cells[i]) {
			if !withinSearchBox(s.Lat, s.Lon, l.lat, l.lon) {
				continue
			}
			if m := metersBetween(s.Lat, s.Lon, l.lat, l.lon); m < nearest {
				nearest = m
				nearestAttr = l.attribution
			}
		}

		if nearest <= LitRadiusM {
			lit[i] = true
			if _, ok := seen[nearestAttr]; !ok {
				seen[nearestAttr] = struct{}{}
				attributions = append(attributions, nearestAttr)
			}
		}
	}

	sort.Slice(attributions, func(i, j int) bool {
		if attributions[i].Text != attributions[j].Text {
			return attributions[i].Text < attributions[j].Text
		}
		return attributions[i].Href < attributions[j].Href
	})
	if attributions == nil {
		attributions = []Attribution{}
	}
	return lit, attributions
}

// buildStretches converts per-sample lit flags into maximal lit stretches. A
// stretch starts and ends halfway to the adjacent unlit sample, clipped to
// the track ends, so a lit sample at distance d is treated as lit over
// [d - SampleStepM/2, d + SampleStepM/2]. It returns the stretches in
// ascending order and the total lit distance.
func buildStretches(samples []track.InterpolatedPoint, lit []bool) ([]LitStretch, float64) {
	stretches := []LitStretch{}
	litDistanceM := 0.0
	for i := 0; i < len(samples); i++ {
		if !lit[i] {
			continue
		}
		j := i
		for j+1 < len(samples) && lit[j+1] {
			j++
		}

		start := samples[i].DistanceM
		if i > 0 {
			start = (samples[i-1].DistanceM + samples[i].DistanceM) / 2
		}
		end := samples[j].DistanceM
		if j < len(samples)-1 {
			end = (samples[j].DistanceM + samples[j+1].DistanceM) / 2
		}

		stretches = append(stretches, LitStretch{StartDistanceM: start, EndDistanceM: end})
		litDistanceM += end - start
		i = j
	}
	return stretches, litDistanceM
}

// withinSearchBox reports whether a lamp could be within [LitRadiusM] of a
// sample, by comparing their offsets in an equirectangular approximation of
// the local tangent plane. It never rejects a pair whose great-circle
// distance is within [LitRadiusM], but it avoids the cgo distance call for
// all pairs that are clearly farther apart.
func withinSearchBox(sampleLat, sampleLon, lampLat, lampLon float64) bool {
	if math.Abs((lampLat-sampleLat)*metersPerDegreeLat) > searchBoxHalfWidthM {
		return false
	}
	dLonM := (lampLon - sampleLon) * metersPerDegreeLat * math.Cos(sampleLat*math.Pi/180)
	return math.Abs(dLonM) <= searchBoxHalfWidthM
}

// metersBetween returns the great-circle distance in meters between two WGS84
// coordinates.
func metersBetween(lat1, lon1, lat2, lon2 float64) float64 {
	a := track.Point{Lat: lat1, Lon: lon1}
	b := track.Point{Lat: lat2, Lon: lon2}
	return a.MetersTo(&b)
}
