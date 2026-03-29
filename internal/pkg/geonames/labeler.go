package geonames

import (
	"bytes"
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/geonames/cols"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/load"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/track"
)

const (
	// bboxPadDeg is the padding in degrees added to the track bounding box (~5 km).
	bboxPadDeg = 0.05

	// maxPlaceDistM is the maximum distance from track for populated places.
	maxPlaceDistM = 3000

	// maxLandmarkDistM is the maximum distance from track for landmarks.
	maxLandmarkDistM = 5000

	// polylineSubsampleM is the distance between polyline samples for geometry ops.
	polylineSubsampleM = 500

	// minSubdivisionPop is the minimum population for a city subdivision (PPLX)
	// to be considered as a candidate. Subdivisions below this threshold are
	// skipped because they clutter labels with neighborhood names. Only very
	// large subdivisions (e.g. boroughs of major cities) pass this filter.
	minSubdivisionPop = 100000

	// maxLabelLen caps the formatted label length in bytes.
	maxLabelLen = 80

	// maxWaypoints is the absolute cap on selected waypoints in a label.
	maxWaypoints = 7
)

type candidateKind int

const (
	candidatePlace candidateKind = iota
	candidateLandmark
)

// candidate is a geoname entry that lies near the track, annotated with
// its distance to the track and its position along it.
type candidate struct {
	name        string
	countryCode string
	admin1Code  string
	featureCode string
	population  int64
	lat, lon    float64
	kind        candidateKind

	trackDist float64 // Minimum distance to track polyline in meters.
	trackFrac float64 // Position along track as fraction 0.0 to 1.0.
	score     float64

	// Resolved after selection.
	admin1Name string
}

// LabelerArgs are the arguments for the track geoname labeler job.
type LabelerArgs struct {
	TrackID string `json:"trackId"`
}

// Kind implements [jobs.Args].
func (LabelerArgs) Kind() string { return "geonames.labeler" }

var _ jobs.Args = (*LabelerArgs)(nil)

// Labeler generates a geoname label for a track by querying nearby places
// and landmarks, scoring them by relevance, and selecting the most notable
// ones. Use [NewLabeler] to create an instance.
type Labeler struct {
	d *db.DB
}

// NewLabeler creates a new [Labeler] instance.
func NewLabeler(d *db.DB) *Labeler {
	return &Labeler{d: d}
}

var _ jobs.Job[LabelerArgs] = (*Labeler)(nil)

// Run implements [jobs.Job].
// It loads the track, queries all nearby populated places and landmarks in the
// track corridor, scores them by relevance (population, proximity, feature type),
// selects the most notable ones with adaptive suppression, and stores a generated
// label in track_geonames.
func (l *Labeler) Run(ctx context.Context, args LabelerArgs) error {
	t, err := l.d.QueryRO().GetTrackByUUID(ctx, args.TrackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logg.Info(ctx, "track not found, skipping labeling", "trackId", args.TrackID)
			return nil
		}
		return fmt.Errorf("get track: %w", err)
	}

	b, err := blob.Get(ctx, l.d.QueryRO(), t.BlobID)
	if err != nil {
		return fmt.Errorf("get blob: %w", err)
	}

	src, err := load.Blob(t.OriginalFilename, bytes.NewReader(b.Content))
	if err != nil {
		return fmt.Errorf("parse blob: %w", err)
	}

	tr, err := track.New(src, 0)
	if err != nil {
		logg.Debug(ctx, "track has fewer than 2 points, skipping labeling", "trackId", args.TrackID)
		return nil
	}
	pts := tr.Points()

	polyline := pts.Subsample(polylineSubsampleM)
	bbox := trackBBox(pts, bboxPadDeg)
	budget := labelBudget(t.TotalDistanceM / 1000)

	candidates := l.gatherCandidates(ctx, bbox, polyline)
	if len(candidates) == 0 {
		logg.Debug(ctx, "no geoname candidates for track", "trackId", args.TrackID)
		return nil
	}

	for i := range candidates {
		candidates[i].score = scoreCandidate(&candidates[i])
	}

	selected := selectCandidates(candidates, budget)
	l.resolveAdmin1Names(ctx, selected)

	label := formatLabel(selected)
	if label == "" {
		logg.Debug(ctx, "no geoname results for track", "trackId", args.TrackID)
		return nil
	}

	err = l.d.QueryRW().UpsertTrackGeoname(ctx, db.UpsertTrackGeonameParams{
		TrackID:   args.TrackID,
		Label:     label,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("upsert track geoname: %w", err)
	}

	logg.Info(ctx, "track labeled", "trackId", args.TrackID, "label", label)
	return nil
}

// gatherCandidates queries populated places and terrain landmarks within the
// track corridor and computes their distance and position along the track.
func (l *Labeler) gatherCandidates(ctx context.Context, bbox [4]float64, polyline track.Points) []candidate {
	bboxParams := db.FindPlacesInBBoxParams{
		MinLat: bbox[0], MaxLat: bbox[1],
		MinLon: bbox[2], MaxLon: bbox[3],
	}

	var candidates []candidate

	places, err := l.d.QueryRO().FindPlacesInBBox(ctx, bboxParams)
	if err != nil {
		logg.Debug(ctx, "find places in bbox failed", "err", err)
	}
	for _, p := range places {
		if skipFeatureCode(p.FeatureCode, p.Population) {
			continue
		}
		pt := track.Point{Lat: p.Latitude, Lon: p.Longitude}
		dist, frac := minDistToPolyline(pt, polyline)
		if dist > maxPlaceDistM {
			continue
		}
		candidates = append(candidates, candidate{
			name:        p.Name,
			countryCode: p.CountryCode,
			admin1Code:  p.Admin1Code,
			featureCode: p.FeatureCode,
			population:  p.Population,
			lat:         p.Latitude,
			lon:         p.Longitude,
			kind:        candidatePlace,
			trackDist:   dist,
			trackFrac:   frac,
		})
	}

	landmarks, err := l.d.QueryRO().FindLandmarksInBBox(ctx, db.FindLandmarksInBBoxParams{
		MinLat: bbox[0], MaxLat: bbox[1],
		MinLon: bbox[2], MaxLon: bbox[3],
	})
	if err != nil {
		logg.Debug(ctx, "find landmarks in bbox failed", "err", err)
	}
	for _, lm := range landmarks {
		pt := track.Point{Lat: lm.Latitude, Lon: lm.Longitude}
		dist, frac := minDistToPolyline(pt, polyline)
		if dist > maxLandmarkDistM {
			continue
		}
		candidates = append(candidates, candidate{
			name:        lm.Name,
			countryCode: lm.CountryCode,
			admin1Code:  lm.Admin1Code,
			featureCode: lm.FeatureCode,
			lat:         lm.Latitude,
			lon:         lm.Longitude,
			kind:        candidateLandmark,
			trackDist:   dist,
			trackFrac:   frac,
		})
	}

	return candidates
}

// skipFeatureCode returns true if the feature code should be excluded from
// candidates. City subdivisions (PPLX), abandoned places (PPLQ), and destroyed
// places (PPLW) are skipped. Subdivisions are kept only if their population
// exceeds minSubdivisionPop, indicating a major borough (e.g. Manhattan).
func skipFeatureCode(code string, population int64) bool {
	switch code {
	case cols.FeatureCodePPLQ, cols.FeatureCodePPLW:
		return true
	case cols.FeatureCodePPLX:
		return population < minSubdivisionPop
	}
	return false
}

// resolveAdmin1Names looks up the admin1 name for each selected candidate.
func (l *Labeler) resolveAdmin1Names(ctx context.Context, selected []candidate) {
	for i := range selected {
		c := &selected[i]
		if c.admin1Code == "" {
			continue
		}
		code := c.countryCode + "." + c.admin1Code
		row, err := l.d.QueryRO().GetGeonameAdmin1(ctx, code)
		if err != nil {
			continue
		}
		c.admin1Name = row.Name
	}
}

// labelBudget returns the maximum number of waypoints for a given track
// distance in kilometers.
func labelBudget(trackDistKm float64) int {
	budget := 2 + int(trackDistKm/30)
	return min(budget, maxWaypoints)
}

// scoreCandidate computes a relevance score for a candidate. Higher is better.
func scoreCandidate(c *candidate) float64 {
	score := 0.0

	if c.kind == candidatePlace && c.population > 0 {
		score += math.Log10(float64(c.population) + 1)
	}

	maxDist := float64(maxPlaceDistM)
	if c.kind == candidateLandmark {
		maxDist = float64(maxLandmarkDistM)
	}
	proximity := 1.0 - (c.trackDist / maxDist)
	score += proximity * 2.0

	switch c.featureCode {
	case cols.FeatureCodePASS:
		score += 3.0
	case cols.FeatureCodePK, cols.FeatureCodeMT:
		score += 2.0
	}

	return score
}

// suppressionRadius returns the track-fraction radius within which a selected
// candidate suppresses lower-scoring candidates. Larger cities suppress over a
// wider range, while landmarks have a small radius.
func suppressionRadius(c *candidate) float64 {
	if c.kind == candidateLandmark {
		return 0.03
	}
	if c.population > 100000 {
		return 0.15
	}
	if c.population > 10000 {
		return 0.10
	}
	return 0.05
}

// selectCandidates picks the most relevant candidates using greedy selection
// with adaptive suppression. The start and end areas of the track are
// guaranteed representation. Results are sorted by track fraction.
func selectCandidates(all []candidate, budget int) []candidate {
	if len(all) == 0 {
		return nil
	}

	slices.SortFunc(all, func(a, b candidate) int {
		return cmp.Compare(b.score, a.score)
	})

	// Pre-select best candidates near start and end.
	var selected []candidate
	startIdx := -1
	endIdx := -1
	for i, c := range all {
		if startIdx == -1 && c.trackFrac <= 0.1 {
			startIdx = i
		}
		if endIdx == -1 && c.trackFrac >= 0.9 {
			endIdx = i
		}
	}
	if startIdx >= 0 {
		selected = append(selected, all[startIdx])
	}
	if endIdx >= 0 && endIdx != startIdx {
		selected = append(selected, all[endIdx])
	}

	for _, c := range all {
		if len(selected) >= budget {
			break
		}
		if isSuppressed(c, selected) {
			continue
		}
		selected = append(selected, c)
	}

	slices.SortFunc(selected, func(a, b candidate) int {
		return cmp.Compare(a.trackFrac, b.trackFrac)
	})

	return selected
}

// isSuppressed returns true if c is within the suppression radius of any
// already-selected candidate. Landmarks are only suppressed by other landmarks,
// not by populated places, since they represent distinct points of interest.
func isSuppressed(c candidate, selected []candidate) bool {
	for _, s := range selected {
		if c.kind == candidateLandmark && s.kind != candidateLandmark {
			continue
		}
		radius := suppressionRadius(&s)
		if math.Abs(c.trackFrac-s.trackFrac) < radius {
			return true
		}
	}
	return false
}

// formatLabel produces a one-line summary from a list of selected candidates,
// ordered by track fraction. Intermediate waypoints are included if they fit
// within maxLabelLen. Examples:
//
//	"Zurich, Zurich, CH"
//	"Zurich - Bern, CH"
//	"Zurich - Grimselpass - Bern, CH"
func formatLabel(selected []candidate) string {
	if len(selected) == 0 {
		return ""
	}

	first := selected[0]
	last := selected[len(selected)-1]
	suffix := formatSuffix(first, last)

	// Collect all names.
	names := make([]string, len(selected))
	for i, c := range selected {
		names[i] = c.name
	}

	// Try full label with all intermediates, trimming from the middle if too long.
	for len(names) > 2 {
		label := strings.Join(names, " - ") + suffix
		if len(label) <= maxLabelLen {
			return label
		}
		// Drop the lowest-scored intermediate (not first or last).
		dropIdx := lowestScoredIntermediate(selected, names)
		names = slices.Delete(names, dropIdx, dropIdx+1)
		selected = slices.Delete(selected, dropIdx, dropIdx+1)
	}

	return strings.Join(names, " - ") + suffix
}

// lowestScoredIntermediate returns the index of the intermediate candidate
// (not first or last) with the lowest score.
func lowestScoredIntermediate(selected []candidate, names []string) int {
	minIdx := 1
	for i := 1; i < len(names)-1; i++ {
		if selected[i].score < selected[minIdx].score {
			minIdx = i
		}
	}
	return minIdx
}

// formatSuffix returns the country/region suffix for a label given its
// first and last candidates.
func formatSuffix(first, last candidate) string {
	if first.countryCode == last.countryCode && first.countryCode != "" {
		if first.admin1Name != "" && first.admin1Name == last.admin1Name {
			return ", " + first.admin1Name + ", " + first.countryCode
		}
		return ", " + first.countryCode
	}
	if first.countryCode != "" {
		s := " (" + first.countryCode
		if last.countryCode != "" && last.countryCode != first.countryCode {
			s += "/" + last.countryCode
		}
		s += ")"
		return s
	}
	return ""
}

// trackBBox computes the bounding box of a set of track points with padding
// in degrees. Returns [minLat, maxLat, minLon, maxLon].
func trackBBox(pts track.Points, padDeg float64) [4]float64 {
	minLat, maxLat := pts[0].Lat, pts[0].Lat
	minLon, maxLon := pts[0].Lon, pts[0].Lon
	for _, p := range pts[1:] {
		minLat = min(minLat, p.Lat)
		maxLat = max(maxLat, p.Lat)
		minLon = min(minLon, p.Lon)
		maxLon = max(maxLon, p.Lon)
	}
	return [4]float64{minLat - padDeg, maxLat + padDeg, minLon - padDeg, maxLon + padDeg}
}

// minDistToPolyline computes the minimum distance in meters from pt to the
// polyline defined by pts, and returns the approximate track fraction (0.0-1.0)
// where the closest point lies.
func minDistToPolyline(pt track.Point, pts track.Points) (distM float64, frac float64) {
	if len(pts) < 2 {
		d := pt.MetersTo(&pts[0])
		return d, 0
	}

	bestDist := math.MaxFloat64
	bestCumDist := 0.0
	cumDist := 0.0

	for i := 1; i < len(pts); i++ {
		segLen := pts[i-1].MetersTo(&pts[i])
		d, t := pointToSegmentDist(pt, pts[i-1], pts[i])
		if d < bestDist {
			bestDist = d
			bestCumDist = cumDist + t*segLen
		}
		cumDist += segLen
	}

	if cumDist == 0 {
		return bestDist, 0
	}
	return bestDist, bestCumDist / cumDist
}

// pointToSegmentDist returns the shortest distance in meters from p to the
// line segment a-b, and the interpolation parameter t (0.0 at a, 1.0 at b)
// of the closest point on the segment.
func pointToSegmentDist(p, a, b track.Point) (distM float64, t float64) {
	// Project p onto the line a-b using a flat-earth approximation for the
	// parameter t, then use great-circle distance for the actual result.
	dx := b.Lon - a.Lon
	dy := b.Lat - a.Lat
	lenSq := dx*dx + dy*dy

	if lenSq == 0 {
		return p.MetersTo(&a), 0
	}

	t = ((p.Lon-a.Lon)*dx + (p.Lat-a.Lat)*dy) / lenSq
	t = max(0, min(1, t))

	closest := track.Point{
		Lat: a.Lat + t*(b.Lat-a.Lat),
		Lon: a.Lon + t*(b.Lon-a.Lon),
	}
	return p.MetersTo(&closest), t
}
