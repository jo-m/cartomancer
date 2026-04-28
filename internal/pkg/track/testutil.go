package track

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// loadGPXPoints parses a GPX file and returns the track points.
// Standalone here because the load package imports [track].
func loadGPXPoints(t *testing.T, path string) Points {
	t.Helper()

	type trkpt struct {
		Lat float64 `xml:"lat,attr"`
		Lon float64 `xml:"lon,attr"`
		Ele float64 `xml:"ele"`
	}
	var gpx struct {
		Tracks []struct {
			Segments []struct {
				Points []trkpt `xml:"trkpt"`
			} `xml:"trkseg"`
		} `xml:"trk"`
	}

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	require.NoError(t, xml.NewDecoder(f).Decode(&gpx))

	var pts Points
	for _, trk := range gpx.Tracks {
		for _, seg := range trk.Segments {
			for _, p := range seg.Points {
				pts = append(pts, Point{Lat: p.Lat, Lon: p.Lon, Elevation: p.Ele})
			}
		}
	}

	cumDist := 0.0
	for i := 1; i < len(pts); i++ {
		cumDist += pts[i-1].MetersTo(&pts[i])
		pts[i].Distance = cumDist
	}

	return pts
}
