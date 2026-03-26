package load

import (
	"encoding/xml"
	"fmt"
	"io"
	"iter"
	"strings"
	"time"

	"jo-m.ch/go/detour/internal/pkg/track"
)

type Link struct {
	Href string `xml:"href,attr"`
	Text string `xml:"text"`
	Type string `xml:"type"`
}

type Author struct {
	Name string `xml:"name"`
	Link *Link  `xml:"link"`
}

type Metadata struct {
	Time   *time.Time `xml:"time"`
	Name   *string    `xml:"name"`
	Link   *Link      `xml:"link"`
	Author *Author    `xml:"author"`
}

type TrackPoint struct {
	Lat  float64    `xml:"lat,attr"`
	Lon  float64    `xml:"lon,attr"`
	Ele  float64    `xml:"ele"`
	Time *time.Time `xml:"time"`
}

type TrackSegment struct {
	Points []TrackPoint `xml:"trkpt"`
}

type Track struct {
	Name        string         `xml:"name"`
	Description string         `xml:"desc"`
	Type        string         `xml:"type"`
	Segments    []TrackSegment `xml:"trkseg"`
}

type Waypoint struct {
	Lat         float64 `xml:"lat,attr"`
	Lon         float64 `xml:"lon,attr"`
	Name        string  `xml:"name"`
	Description string  `xml:"desc"`
	Sym         string  `xml:"sym"`
	Type        string  `xml:"type"`
}

type GPX struct {
	filename string

	XMLName   xml.Name   `xml:"gpx"`
	Creator   string     `xml:"creator,attr"`
	XMetadata Metadata   `xml:"metadata"`
	Tracks    []Track    `xml:"trk"`
	Waypoints []Waypoint `xml:"wpt"`
}

// Compile time interface check.
var _ track.TrackSource = (*GPX)(nil)

func loadGpx(filename string, contents io.Reader) (track.TrackSource, error) {
	g := GPX{
		filename: filename,
	}
	err := xml.NewDecoder(contents).Decode(&g)
	if err != nil {
		return nil, fmt.Errorf("error decoding XML: %v", err)
	}

	if len(g.Tracks) != 1 {
		return nil, fmt.Errorf("expected 1 track in file, found %d", len(g.Tracks))
	}

	return &g, nil
}

// detectTrackType infers whether a GPX track was recorded or planned.
func (g *GPX) detectTrackType() track.TrackType {
	hasAuthor := g.XMetadata.Author != nil && g.XMetadata.Author.Name != ""
	hasAuthorLink := g.XMetadata.Author != nil && g.XMetadata.Author.Link != nil && g.XMetadata.Author.Link.Href != ""
	hasMetadataName := g.XMetadata.Name != nil && *g.XMetadata.Name != ""
	hasTrackName := len(g.Tracks) > 0 && g.Tracks[0].Name != ""

	isGarmin := g.XMetadata.Link != nil && strings.Contains(g.XMetadata.Link.Href, "garmin.com")

	if isGarmin {
		// Garmin Connect: recorded tracks have a name in <trk> only, planned
		// tracks have it in both <metadata> and <trk>.
		if hasTrackName && !hasMetadataName {
			return track.TrackTypeRecorded
		}
		return track.TrackTypePlanned
	}

	// Strava: tracks without author name/URL and without a source name are
	// recorded. This heuristic holds for tracks exported from Strava only.
	if !hasAuthor && !hasAuthorLink {
		return track.TrackTypeRecorded
	}

	return track.TrackTypePlanned
}

// Metadata returns extracted track metadata from the GPX file.
func (g *GPX) Metadata() track.Metadata {
	ret := track.Metadata{
		Source:    g.Creator,
		TrackType: g.detectTrackType(),
	}

	t := g.Tracks[0]

	ret.Name = t.Name
	ret.Description = t.Description

	switch strings.ToLower(t.Type) {
	case "road_biking": // Garmin.
		ret.Sport = track.SportCycling
		ret.SubSport = track.SubSportCyclingRoad
	case "running": // Garmin.
		ret.Sport = track.SportRunning
		ret.SubSport = track.SubSportRunningOutdoor
	case "cycling": // Strava.
		ret.Sport = track.SportCycling
		ret.SubSport = track.SubSportCyclingRoad
	case "ride": // Strava.
		ret.Sport = track.SportCycling
		ret.SubSport = track.SubSportCyclingRoad
	case "gravel_biking": // Strava.
		ret.Sport = track.SportCycling
		ret.SubSport = track.SubSportCyclingGravel
	default:
		ret.Sport = track.SportUnknown
		ret.SubSport = track.SubSportUnknown
	}

	if g.XMetadata.Time != nil {
		ret.OriginalCreatedAt = g.XMetadata.Time
	}

	// Strava route.
	if g.XMetadata.Link != nil {
		ret.LinkURL = g.XMetadata.Link.Href
	}
	if g.XMetadata.Author != nil {
		ret.Author = g.XMetadata.Author.Name
		if g.XMetadata.Author.Link != nil {
			ret.AuthorLinkURL = g.XMetadata.Author.Link.Href
		}
	}

	return ret
}

func (g *GPX) All() iter.Seq[track.Point] {
	return func(yield func(track.Point) bool) {
		for _, trk := range g.Tracks {
			for _, seg := range trk.Segments {
				for _, pt := range seg.Points {
					point := track.Point{
						Lat:       pt.Lat,
						Lon:       pt.Lon,
						Elevation: pt.Ele,
					}
					if pt.Time != nil {
						point.Time = *pt.Time
					}

					if !yield(point) {
						return
					}
				}
			}
		}
	}
}
