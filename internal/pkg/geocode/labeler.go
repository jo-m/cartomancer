package geocode

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/load"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/track"
)

const (
	// searchRadiusDeg is the bounding box half-width in degrees (~10 km).
	searchRadiusDeg = 0.1

	// sampleDistM is the minimum distance between sampled points in meters.
	sampleDistM = 5000

	// maxSamples caps how many points are looked up.
	maxSamples = 10
)

// LabelerArgs are the arguments for the track geoname labeler job.
type LabelerArgs struct {
	TrackID string `json:"trackId"`
}

// Kind implements [jobs.Args].
func (LabelerArgs) Kind() string { return "geocode.labeler" }

var _ jobs.Args = (*LabelerArgs)(nil)

// Labeler generates a geoname label for a track by sampling points and
// looking up nearby populated places. Use [NewLabeler] to create an instance.
type Labeler struct {
	d *db.DB
}

// NewLabeler creates a new [Labeler] instance.
func NewLabeler(d *db.DB) *Labeler {
	return &Labeler{d: d}
}

var _ jobs.Job[LabelerArgs] = (*Labeler)(nil)

// Run implements [jobs.Job].
// It loads the track, samples points along it, looks up the nearest populated
// place for each sample, and stores a generated label in track_geonames.
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

	tr := track.New(src, 0)
	pts := tr.Points().Subsample(sampleDistM)
	if len(pts) > maxSamples {
		pts = evenSample(pts, maxSamples)
	}

	label := l.buildLabel(ctx, pts)
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

// place holds a resolved geoname for deduplication.
type place struct {
	Name        string
	Admin1Name  string
	CountryCode string
}

// buildLabel looks up the nearest populated place for each sample point
// and builds a concise label string like "Zurich - Bern, CH".
func (l *Labeler) buildLabel(ctx context.Context, pts track.Points) string {
	var places []place
	seen := make(map[string]bool)

	for _, pt := range pts {
		row, err := l.d.QueryRO().NearestPlace(ctx, db.NearestPlaceParams{
			Lat:    pt.Lat,
			Lon:    pt.Lon,
			MinLat: pt.Lat - searchRadiusDeg,
			MaxLat: pt.Lat + searchRadiusDeg,
			MinLon: pt.Lon - searchRadiusDeg,
			MaxLon: pt.Lon + searchRadiusDeg,
		})
		if err != nil {
			continue
		}

		if seen[row.Name] {
			continue
		}
		seen[row.Name] = true

		admin1 := l.resolveAdmin1(ctx, row.CountryCode, row.Admin1Code)
		places = append(places, place{
			Name:        row.Name,
			Admin1Name:  admin1,
			CountryCode: row.CountryCode,
		})
	}

	return formatLabel(places)
}

// resolveAdmin1 looks up the admin1 name for a country+admin1 code pair.
func (l *Labeler) resolveAdmin1(ctx context.Context, countryCode, admin1Code string) string {
	if admin1Code == "" {
		return ""
	}
	code := countryCode + "." + admin1Code
	row, err := l.d.QueryRO().GetGeonameAdmin1(ctx, code)
	if err != nil {
		return ""
	}
	return row.Name
}

// formatLabel produces a one-line summary from a list of resolved places.
// Examples: "Zurich - Bern, ZH, CH", "Paris, Ile-de-France, FR".
func formatLabel(places []place) string {
	if len(places) == 0 {
		return ""
	}

	// Collect unique place names in order.
	var names []string
	for _, p := range places {
		names = append(names, p.Name)
	}

	// Use first and last place's context for the suffix.
	first := places[0]
	last := places[len(places)-1]

	var label string
	if len(names) == 1 {
		label = names[0]
	} else if len(names) == 2 {
		label = names[0] + " - " + names[1]
	} else {
		label = names[0] + " - " + names[len(names)-1]
	}

	// Add region context if both endpoints share the same country.
	if first.CountryCode == last.CountryCode && first.CountryCode != "" {
		if first.Admin1Name != "" && first.Admin1Name == last.Admin1Name {
			label += ", " + first.Admin1Name
		}
		label += ", " + first.CountryCode
	} else {
		// Different countries: append both.
		if first.CountryCode != "" {
			label += " (" + first.CountryCode
			if last.CountryCode != "" && last.CountryCode != first.CountryCode {
				label += "/" + last.CountryCode
			}
			label += ")"
		}
	}

	return label
}

// evenSample picks n evenly-spaced points from pts, always including first and last.
func evenSample(pts track.Points, n int) track.Points {
	if len(pts) <= n {
		return pts
	}
	result := make(track.Points, 0, n)
	for i := range n {
		idx := i * (len(pts) - 1) / (n - 1)
		result = append(result, pts[idx])
	}
	return result
}
