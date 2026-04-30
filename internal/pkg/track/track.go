package track

import (
	"fmt"
	"iter"
	"time"
)

// Sport identifies the primary sport of a track.
type Sport int

// When editing this list, the frontend code at frontend/src/lib/sports.ts MUST be updated as well.
const (
	SportUnknown Sport = iota
	SportRunning
	SportCycling
)

// SubSport further classifies the sport variant of a track.
type SubSport int

// When editing this list, the frontend code at frontend/src/lib/sports.ts MUST be updated as well.
const (
	SubSportUnknown SubSport = iota
	SubSportRunningOutdoor
	SubSportRunningTreadmill

	SubSportCyclingRoad
	SubSportCyclingSpinning
	SubSportCyclingIndoorCycling
	SubSportCyclingMountain
	SubSportCyclingGravel
	SubSportCyclingCommuting
)

// TrackType distinguishes planned routes from recorded activities.
//
//revive:disable-next-line:exported
type TrackType int

// TrackTypeUnknown, TrackTypePlanned, and TrackTypeRecorded enumerate the known track types.
const (
	TrackTypeUnknown TrackType = iota
	TrackTypePlanned
	TrackTypeRecorded
)

// FileFormat identifies the file format of the original track file.
type FileFormat int

// FileFormatGPX and FileFormatFIT enumerate the supported file formats.
const (
	FileFormatGPX FileFormat = iota
	FileFormatFIT
)

// Metadata holds descriptive information about a track extracted from its source file.
type Metadata struct {
	Name          string
	Description   string
	Source        string
	Author        string
	AuthorLinkURL string

	TrackType TrackType
	// URL to this track on the platform it comes from (e.g. Komoot, Strava, ...).
	LinkURL string

	Sport    Sport
	SubSport SubSport

	TotalDistanceM float64
	TotalAscentM   float64

	MinElevationM *float64
	MaxElevationM *float64

	StartLat, StartLon *float64
	EndLat, EndLon     *float64

	BoundsMinLat, BoundsMinLon *float64
	BoundsMaxLat, BoundsMaxLon *float64

	// OriginalCreatedAt is the recording/creation timestamp from the original file.
	OriginalCreatedAt *time.Time
}

// Track holds a parsed GPS track with at least two points.
// Use [New] to construct; it enforces the minimum-points invariant.
type Track struct {
	meta Metadata
	// The track points as loaded from the original source.
	pts []Point
}

// Len returns the number of points in the track.
func (t *Track) Len() int {
	return len(t.pts)
}

// Points returns the track's point sequence.
func (t *Track) Points() Points {
	return Points(t.pts)
}

func (t *Track) computeDistanceM() float64 {
	distM := 0.
	for i, p1 := range t.pts[1:] {
		p0 := t.pts[i]
		distM += p1.MetersTo(&p0)
	}
	return float64(distM)
}

func (t *Track) computeAscentM() float64 {
	ascM := 0.
	for i, p1 := range t.pts[1:] {
		p0 := t.pts[i]
		a := p1.Elevation - p0.Elevation
		if a > 0 {
			ascM += a
		}
	}
	return float64(ascM)
}

func (t *Track) computeElevationBounds() (minM, maxM *float64) {
	if len(t.pts) == 0 {
		return nil, nil
	}
	lo, hi := t.pts[0].Elevation, t.pts[0].Elevation
	for _, p := range t.pts[1:] {
		if p.Elevation < lo {
			lo = p.Elevation
		}
		if p.Elevation > hi {
			hi = p.Elevation
		}
	}
	return &lo, &hi
}

const (
	defaultBikeSubSport = SubSportCyclingRoad
	defaultRunSubSport  = SubSportRunningOutdoor
)

// EnhancedMetadata returns Metadata with some additional guessed/computed information.
func (t *Track) EnhancedMetadata() Metadata {
	ret := t.meta

	if ret.TotalDistanceM == 0 {
		ret.TotalDistanceM = t.computeDistanceM()
	}

	if ret.TotalAscentM == 0 {
		ret.TotalAscentM = t.computeAscentM()
	}

	if ret.SubSport == SubSportUnknown {
		switch ret.Sport {
		case SportCycling:
			ret.SubSport = defaultBikeSubSport
		case SportRunning:
			ret.SubSport = defaultRunSubSport
		}
	}

	ret.MinElevationM, ret.MaxElevationM = t.computeElevationBounds()

	if len(t.pts) > 0 {
		first := t.pts[0]
		last := t.pts[len(t.pts)-1]
		ret.StartLat = &first.Lat
		ret.StartLon = &first.Lon
		ret.EndLat = &last.Lat
		ret.EndLon = &last.Lon

		minLat, maxLat := t.pts[0].Lat, t.pts[0].Lat
		minLon, maxLon := t.pts[0].Lon, t.pts[0].Lon
		for _, p := range t.pts[1:] {
			minLat = min(minLat, p.Lat)
			maxLat = max(maxLat, p.Lat)
			minLon = min(minLon, p.Lon)
			maxLon = max(maxLon, p.Lon)
		}
		ret.BoundsMinLat = &minLat
		ret.BoundsMinLon = &minLon
		ret.BoundsMaxLat = &maxLat
		ret.BoundsMaxLon = &maxLon
	}

	return ret
}

// PreviewSVG generates a square SVG preview image of the track.
// opts controls the canvas size, stroke width, and color.
// If bounds is non-nil, its extents are used directly instead of being computed from the track points.
func (t *Track) PreviewSVG(opts PreviewOptions, bounds *Bounds) string {
	return Points(t.pts).PreviewSVG(opts, bounds)
}

// ProfileSVG generates an altitude profile SVG for the track.
// opts controls the canvas width, stroke width, and color.
// The canvas height is opts.Size/4; the Y axis is fixed.
func (t *Track) ProfileSVG(opts PreviewOptions) string {
	return Points(t.pts).ProfileSVG(opts)
}

// TrackSource is a source of track points and metadata, typically a parsed GPX or FIT file.
//
//revive:disable-next-line:exported
type TrackSource interface {
	Metadata() Metadata
	All() iter.Seq[Point]
}

// New creates a Track from a TrackSource. Returns an error if the source
// contains fewer than two points.
func New(src TrackSource) (*Track, error) {
	pts := []Point{}
	for p := range src.All() {
		pts = append(pts, p)
	}

	if len(pts) < 2 {
		return nil, fmt.Errorf("track must have at least 2 points, got %d", len(pts))
	}

	cumDist := 0.0
	for i := 1; i < len(pts); i++ {
		cumDist += pts[i-1].MetersTo(&pts[i])
		pts[i].Distance = cumDist
	}

	track := Track{
		meta: src.Metadata(),
		pts:  pts,
	}

	return &track, nil
}
