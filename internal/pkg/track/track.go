package track

import (
	"iter"
	"time"
)

type Sport int

const (
	SportUnknown Sport = iota
	SportRunning
	SportCycling
)

type SubSport int

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

type TrackType int

const (
	TrackTypeUnknown TrackType = iota
	TrackTypePlanned
	TrackTypeRecorded
)

type FileFormat int

const (
	FileFormatGPX FileFormat = iota
	FileFormatFIT
)

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

	StartLat, StartLon *float64
	EndLat, EndLon     *float64

	// OriginalCreatedAt is the recording/creation timestamp from the original file.
	OriginalCreatedAt *time.Time
}

type Track struct {
	meta Metadata
	// The track points as loaded from the original source.
	pts []Point
}

func (t *Track) Len() int {
	return len(t.pts)
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

// TODO: Make those configurable per user.
const (
	defaultBikeSubSport = SubSportCyclingRoad
	defaultRunSubSport  = SubSportRunningOutdoor
)

// EnhancedMetadata returns Metadata with some additional guessed information if it is missing from the original.
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

	if len(t.pts) > 0 {
		first := t.pts[0]
		last := t.pts[len(t.pts)-1]
		ret.StartLat = &first.Lat
		ret.StartLon = &first.Lon
		ret.EndLat = &last.Lat
		ret.EndLon = &last.Lon
	}

	return ret
}

// PreviewSVG generates a square SVG preview image of the track at the given pixel size.
func (t *Track) PreviewSVG(size int) string {
	return Points(t.pts).PreviewSVG(size)
}

type TrackSource interface {
	Metadata() Metadata
	All() iter.Seq[Point]
}

func New(src TrackSource, resolution int) *Track {
	pts := []Point{}
	for p := range src.All() {
		pts = append(pts, p)
	}

	track := Track{
		meta: src.Metadata(),
		pts:  pts,
	}

	return &track
}
