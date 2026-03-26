package load

import (
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/track"
)

func ptr(s string) *string { return &s }

func TestDetectTrackType(t *testing.T) {
	tests := []struct {
		name     string
		gpx      GPX
		expected track.TrackType
	}{
		{
			name: "garmin recorded: name in trk only",
			gpx: GPX{
				XMetadata: Metadata{
					Link: &Link{Href: "connect.garmin.com"},
				},
				Tracks: []Track{{Name: "Zurich Road Cycling"}},
			},
			expected: track.TrackTypeRecorded,
		},
		{
			name: "garmin planned: name in both metadata and trk",
			gpx: GPX{
				XMetadata: Metadata{
					Name: ptr("My Route"),
					Link: &Link{Href: "connect.garmin.com"},
				},
				Tracks: []Track{{Name: "My Route"}},
			},
			expected: track.TrackTypePlanned,
		},
		{
			name: "garmin planned: name in metadata only",
			gpx: GPX{
				XMetadata: Metadata{
					Name: ptr("My Route"),
					Link: &Link{Href: "connect.garmin.com"},
				},
				Tracks: []Track{{}},
			},
			expected: track.TrackTypePlanned,
		},
		{
			name: "strava recorded: no author info",
			gpx: GPX{
				XMetadata: Metadata{},
				Tracks:    []Track{{Name: "Morning Ride"}},
			},
			expected: track.TrackTypeRecorded,
		},
		{
			name: "strava planned: has author name and link",
			gpx: GPX{
				XMetadata: Metadata{
					Author: &Author{
						Name: "John Doe",
						Link: &Link{Href: "https://www.strava.com/athletes/12345"},
					},
				},
				Tracks: []Track{{Name: "My Planned Route"}},
			},
			expected: track.TrackTypePlanned,
		},
		{
			name: "strava planned: has author name only",
			gpx: GPX{
				XMetadata: Metadata{
					Author: &Author{Name: "John Doe"},
				},
				Tracks: []Track{{Name: "My Route"}},
			},
			expected: track.TrackTypePlanned,
		},
		{
			name: "strava planned: has author link only",
			gpx: GPX{
				XMetadata: Metadata{
					Author: &Author{
						Link: &Link{Href: "https://www.strava.com/athletes/12345"},
					},
				},
				Tracks: []Track{{Name: "My Route"}},
			},
			expected: track.TrackTypePlanned,
		},
		{
			name: "other source with author: planned",
			gpx: GPX{
				XMetadata: Metadata{
					Author: &Author{
						Link: &Link{Href: "https://www.komoot.de"},
					},
				},
				Tracks: []Track{{Name: "My Route"}},
			},
			expected: track.TrackTypePlanned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.gpx.detectTrackType()
			require.Equal(t, tt.expected, got)
		})
	}
}
