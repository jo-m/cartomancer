package segment

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/cartomancer/internal/pkg/blob"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/load"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

const (
	// maxTrackDistanceM is the maximum total distance of a track to be considered for segmenting.
	maxTrackDistanceM = 200_000
)

// BuilderArgs are the arguments for the [Builder] job.
type BuilderArgs struct{}

// Kind implements [jobs.Args].
func (a BuilderArgs) Kind() string { return "segment.builder" }

var _ jobs.Args = (*BuilderArgs)(nil)

// Builder is a job that extracts shared segments from all tracks globally.
// Submit it with [jobs.Params.Debounce] enabled so that rapid uploads
// are coalesced into a single extraction run.
// Use [NewBuilder] to create a new instance.
type Builder struct {
	d *db.DB
}

// NewBuilder creates a new [Builder] instance.
func NewBuilder(d *db.DB) *Builder {
	return &Builder{d: d}
}

var _ jobs.Job[BuilderArgs] = (*Builder)(nil)

// Run implements [jobs.Job]. It extracts segments from all tracks.
func (b *Builder) Run(ctx context.Context, _ BuilderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	logg.Info(ctx, "segment extraction job started")

	err := BuildSegments(ctx, b.d, DefaultResolution)
	if err != nil {
		return err
	}

	logg.Info(ctx, "segment extraction job finished")
	return nil
}

// BuildSegments loads all tracks, extracts segments, and replaces existing
// segment data in the database.
func BuildSegments(ctx context.Context, d *db.DB, resolution int) error {
	trackCells, err := loadAllTracks(ctx, d, resolution)
	if err != nil {
		return fmt.Errorf("loading tracks: %w", err)
	}
	logg.Info(ctx, "loaded tracks for segment extraction", "tracks", len(trackCells))

	if len(trackCells) < MinTrackCount {
		logg.Info(ctx, "not enough tracks for segment extraction")
		return replaceSegments(ctx, d, nil, resolution)
	}

	// Load track points on-demand for junction refinement and polyline
	// attachment.
	rawLoader := makeRawPointLoader(ctx, d)
	pointLoader := makePointLoader(ctx, d)

	result, err := Extract(trackCells, MinTrackCount, rawLoader)
	if err != nil {
		return fmt.Errorf("extracting segments: %w", err)
	}
	logg.Info(ctx, "extracted segments", "segments", len(result.Segments))

	if err := AttachPolylines(result.Segments, pointLoader, resolution); err != nil {
		return fmt.Errorf("attaching polylines: %w", err)
	}

	return replaceSegments(ctx, d, result.Segments, resolution)
}

// loadAllTracks fetches all groupable tracks and converts them to H3 cells.
func loadAllTracks(ctx context.Context, d *db.DB, resolution int) ([]TrackCells, error) {
	rows, err := d.QueryRO().ListAllGroupableTracks(ctx, maxTrackDistanceM)
	if err != nil {
		return nil, err
	}

	entries := make([]TrackCells, 0, len(rows))
	for _, row := range rows {
		b, err := blob.Get(ctx, d.QueryRO(), row.BlobID)
		if err != nil {
			logg.Error(ctx, "failed to get blob for track, skipping", "trackUUID", row.Uuid, "err", err)
			continue
		}

		src, err := load.Blob(row.OriginalFilename, bytes.NewReader(b.Content))
		if err != nil {
			logg.Error(ctx, "failed to parse track blob, skipping", "trackUUID", row.Uuid, "err", err)
			continue
		}

		cells, err := track.NewCells(src, resolution)
		if err != nil {
			logg.Error(ctx, "failed to build cells, skipping", "trackUUID", row.Uuid, "err", err)
			continue
		}

		entries = append(entries, TrackCells{UUID: row.Uuid, Cells: cells})
	}
	return entries, nil
}

// makeRawPointLoader returns a [RawPointLoader] that loads all GPS points
// for a track from the database without cell indexing.
func makeRawPointLoader(ctx context.Context, d *db.DB) RawPointLoader {
	return func(uuid string) ([]track.Point, error) {
		row, err := d.QueryRO().GetTrackByUUID(ctx, uuid)
		if err != nil {
			return nil, fmt.Errorf("getting track for %s: %w", uuid, err)
		}

		b, err := blob.Get(ctx, d.QueryRO(), row.BlobID)
		if err != nil {
			return nil, fmt.Errorf("getting blob for track %s: %w", uuid, err)
		}

		src, err := load.Blob(row.OriginalFilename, bytes.NewReader(b.Content))
		if err != nil {
			return nil, fmt.Errorf("parsing blob for track %s: %w", uuid, err)
		}

		var pts []track.Point
		for p := range src.All() {
			pts = append(pts, p)
		}
		return pts, nil
	}
}

// makePointLoader returns a [PointLoader] that loads points for a track from
// the database, parsing the blob and mapping each point to its H3 cell.
func makePointLoader(ctx context.Context, d *db.DB) PointLoader {
	return func(uuid string, res int) (map[h3.Cell]track.Point, error) {
		row, err := d.QueryRO().GetTrackByUUID(ctx, uuid)
		if err != nil {
			return nil, fmt.Errorf("getting track for %s: %w", uuid, err)
		}

		b, err := blob.Get(ctx, d.QueryRO(), row.BlobID)
		if err != nil {
			return nil, fmt.Errorf("getting blob for track %s: %w", uuid, err)
		}

		src, err := load.Blob(row.OriginalFilename, bytes.NewReader(b.Content))
		if err != nil {
			return nil, fmt.Errorf("parsing blob for track %s: %w", uuid, err)
		}

		m := make(map[h3.Cell]track.Point)
		for p := range src.All() {
			c := p.Cell(res)
			if _, exists := m[c]; !exists {
				m[c] = p
			}
		}
		return m, nil
	}
}

// replaceSegments deletes all existing segments and inserts new ones.
func replaceSegments(ctx context.Context, d *db.DB, segments []Segment, resolution int) error {
	return d.WithTx(ctx, func(q *db.Queries) error {
		if err := q.DeleteAllSegmentTracks(ctx); err != nil {
			return fmt.Errorf("deleting old segment tracks: %w", err)
		}
		if err := q.DeleteAllSegments(ctx); err != nil {
			return fmt.Errorf("deleting old segments: %w", err)
		}
		if err := q.DeleteAllSegmentJunctions(ctx); err != nil {
			return fmt.Errorf("deleting old junctions: %w", err)
		}

		now := time.Now()

		// Deduplicate junctions by H3 cell.
		junctionUUIDs := make(map[h3.Cell]string)
		ensureJunction := func(j Junction) (string, error) {
			cellStr := j.H3Cell.String()
			if id, ok := junctionUUIDs[j.H3Cell]; ok {
				return id, nil
			}
			id, err := uuid.NewV7()
			if err != nil {
				return "", fmt.Errorf("generating junction UUID: %w", err)
			}
			idStr := id.String()
			if err := q.CreateSegmentJunction(ctx, db.CreateSegmentJunctionParams{
				Uuid:      idStr,
				H3Cell:    cellStr,
				Lat:       j.Lat,
				Lon:       j.Lon,
				CreatedAt: now,
			}); err != nil {
				return "", fmt.Errorf("creating junction: %w", err)
			}
			junctionUUIDs[j.H3Cell] = idStr
			return idStr, nil
		}

		for _, seg := range segments {
			startID, err := ensureJunction(seg.StartJunction)
			if err != nil {
				return err
			}
			endID, err := ensureJunction(seg.EndJunction)
			if err != nil {
				return err
			}

			segID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generating segment UUID: %w", err)
			}

			polyline, err := seg.PolylineJSON()
			if err != nil {
				return fmt.Errorf("encoding polyline: %w", err)
			}

			if err := q.CreateSegment(ctx, db.CreateSegmentParams{
				Uuid:            segID.String(),
				StartJunctionID: startID,
				EndJunctionID:   endID,
				H3Resolution:    int64(resolution),
				DistanceM:       seg.DistanceM,
				AscentM:         0, // H3 cells don't carry elevation.
				NTracks:         int64(len(seg.TrackUUIDs)),
				Polyline:        polyline,
				CreatedAt:       now,
			}); err != nil {
				return fmt.Errorf("creating segment: %w", err)
			}

			for _, trackUUID := range seg.TrackUUIDs {
				if err := q.CreateSegmentTrack(ctx, db.CreateSegmentTrackParams{
					SegmentID: segID.String(),
					TrackID:   trackUUID,
				}); err != nil {
					return fmt.Errorf("creating segment track: %w", err)
				}
			}
		}

		return nil
	})
}
