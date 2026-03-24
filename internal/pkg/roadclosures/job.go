package roadclosures

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

const (
	// jobTimeout is the maximum time the road closures download job may run.
	jobTimeout = 2 * time.Minute

	// jobKind identifies this job in the job queue and in the inserted_by column.
	jobKind = "roadclosures.downloader"

	// CellResolution is the H3 resolution used for coarse spatial indexing (DB lookup).
	// See https://h3geo.org/docs/core-library/restable/.
	CellResolution = 7

	// FineResolution is the H3 resolution used for fine-grained intersection checks.
	FineResolution = 12

	// MinRefreshAge is the minimum time between two successful downloads.
	// The job returns early if the most recent insert is younger than this.
	MinRefreshAge = 23 * time.Hour
)

// DownloaderArgs are the arguments for the road closures downloader job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return jobKind }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader fetches bike road closures from geo.admin.ch and writes them to the database.
// Each run replaces all previously inserted rows in a single transaction.
// Use [NewDownloader] to create an instance.
type Downloader struct {
	d *db.DB
}

// NewDownloader creates a new [Downloader] instance.
func NewDownloader(d *db.DB) *Downloader {
	return &Downloader{d: d}
}

var _ jobs.Job[DownloaderArgs] = (*Downloader)(nil)

// Run implements [jobs.Job].
// It checks whether the data is stale (older than [MinRefreshAge]), fetches
// road closures from the geo.admin.ch API, and in a single transaction deletes
// all rows previously inserted by this job and inserts the freshly downloaded
// ones together with their H3 cell indices.
func (dl *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	// Skip if data is still fresh.
	lastCreated, err := dl.d.QueryRO().GetLatestRoadClosureCreatedAt(ctx, jobKind)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check last run: %w", err)
	}
	if err == nil && time.Since(lastCreated) < MinRefreshAge {
		logg.Info(ctx, "road closures data is recent, skipping download", "lastRun", lastCreated)
		return nil
	}

	logg.Info(ctx, "fetching road closures")
	resp, err := Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch road closures: %w", err)
	}
	logg.Info(ctx, "fetched road closures", "count", len(resp.Results))

	now := time.Now()

	return dl.d.WithTx(ctx, func(tx *db.Queries) error {
		deleted, err := tx.DeleteRoadClosuresByInsertedBy(ctx, jobKind)
		if err != nil {
			return fmt.Errorf("delete old road closures: %w", err)
		}
		logg.Info(ctx, "deleted old road closures", "count", deleted)

		for _, f := range resp.Results {
			if err := insertFeature(ctx, tx, f, now); err != nil {
				return fmt.Errorf("insert feature %d: %w", f.FeatureID, err)
			}
		}

		logg.Info(ctx, "inserted road closures", "count", len(resp.Results))
		return nil
	})
}

// insertFeature inserts a single road closure feature and its H3 cells into the database.
func insertFeature(ctx context.Context, tx *db.Queries, f Feature, now time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate uuid: %w", err)
	}

	geomJSON, err := json.Marshal(f.Geometry)
	if err != nil {
		return fmt.Errorf("marshal geometry: %w", err)
	}

	startsAt, endsAt := parseDurationRange(f.Properties.DurationEn)

	err = tx.InsertRoadClosure(ctx, db.InsertRoadClosureParams{
		Uuid:            id.String(),
		SourceID:        strconv.Itoa(f.FeatureID),
		InsertedBy:      jobKind,
		CreatedAt:       now,
		Type:            f.Properties.SperrungenType,
		StartsAt:        startsAt,
		EndsAt:          endsAt,
		Reason:          nullString(f.Properties.ReasonEn),
		Title:           f.Properties.TitleEn,
		Description:     nullString(f.Properties.AbstractEn),
		ContentProvider: nullString(f.Properties.ContentProviderEn),
		Geometry:        string(geomJSON),
		Attribution:     DataAttribution.Author,
		AttributionHref: DataAttribution.Source,
	})
	if err != nil {
		return fmt.Errorf("insert road closure: %w", err)
	}

	if f.Geometry == nil {
		return nil
	}

	cells := geometryCells(f.Geometry.Geometry(), CellResolution)
	for cell := range cells {
		err = tx.InsertRoadClosureCellRes7(ctx, db.InsertRoadClosureCellRes7Params{
			RoadClosureID: id.String(),
			Cell:          int64(cell),
		})
		if err != nil {
			return fmt.Errorf("insert cell: %w", err)
		}
	}

	return nil
}

// geometryCells computes the set of H3 cells that cover a given orb geometry
// at the specified resolution. Supports Point, MultiPoint, LineString,
// MultiLineString, Polygon, and MultiPolygon.
func geometryCells(g orb.Geometry, resolution int) map[h3.Cell]struct{} {
	cells := make(map[h3.Cell]struct{})
	addPoints(cells, extractPoints(g), resolution)
	return cells
}

// extractPoints recursively collects all coordinate points from a geometry.
func extractPoints(g orb.Geometry) []orb.Point {
	switch v := g.(type) {
	case orb.Point:
		return []orb.Point{v}
	case orb.MultiPoint:
		return []orb.Point(v)
	case orb.LineString:
		return []orb.Point(v)
	case orb.MultiLineString:
		var pts []orb.Point
		for _, ls := range v {
			pts = append(pts, []orb.Point(ls)...)
		}
		return pts
	case orb.Polygon:
		var pts []orb.Point
		for _, ring := range v {
			pts = append(pts, []orb.Point(ring)...)
		}
		return pts
	case orb.MultiPolygon:
		var pts []orb.Point
		for _, poly := range v {
			for _, ring := range poly {
				pts = append(pts, []orb.Point(ring)...)
			}
		}
		return pts
	default:
		return nil
	}
}

// addPoints converts orb.Points to H3 cells and adds them to the set.
// Adjacent points are interpolated at half the hexagon edge length to avoid
// skipping cells on straight segments.
func addPoints(cells map[h3.Cell]struct{}, pts []orb.Point, resolution int) {
	if len(pts) == 0 {
		return
	}

	edgeLenM, err := h3.HexagonEdgeLengthAvgM(resolution)
	if err != nil {
		panic(err)
	}
	stepM := edgeLenM / 2

	for i, pt := range pts {
		cell, err := h3.LatLngToCell(h3.LatLng{Lat: pt.Lat(), Lng: pt.Lon()}, resolution)
		if err != nil {
			continue
		}
		cells[cell] = struct{}{}

		if i == 0 {
			continue
		}

		prev := pts[i-1]
		distM := h3.GreatCircleDistanceM(
			h3.LatLng{Lat: prev.Lat(), Lng: prev.Lon()},
			h3.LatLng{Lat: pt.Lat(), Lng: pt.Lon()},
		)
		steps := int(distM/stepM + 1)
		for j := 1; j < steps; j++ {
			frac := float64(j) / float64(steps)
			lat := prev.Lat() + frac*(pt.Lat()-prev.Lat())
			lon := prev.Lon() + frac*(pt.Lon()-prev.Lon())
			c, err := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lon}, resolution)
			if err != nil {
				continue
			}
			cells[c] = struct{}{}
		}
	}
}

// parseDurationRange extracts start and end dates from a duration string.
// The expected format is "DD.MM.YYYY – DD.MM.YYYY" (en-dash separated).
// Returns null times when the string does not match this pattern.
func parseDurationRange(s string) (startsAt, endsAt sql.NullTime) {
	// The separator is an en-dash (U+2013), optionally surrounded by spaces.
	parts := strings.SplitN(s, "\u2013", 2)
	if len(parts) != 2 {
		return
	}

	const dateLayout = "02.01.2006"

	start, err := time.Parse(dateLayout, strings.TrimSpace(parts[0]))
	if err != nil {
		return
	}
	end, err := time.Parse(dateLayout, strings.TrimSpace(parts[1]))
	if err != nil {
		return
	}

	startsAt = sql.NullTime{Time: start, Valid: true}
	endsAt = sql.NullTime{Time: end, Valid: true}
	return
}

// nullString returns a sql.NullString that is valid only when s is non-empty.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
