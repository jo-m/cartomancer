package streetlights

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

// ErrNilGeometry is returned by [Insert] when called with a nil geometry.
// Streetlights without a geometry can never be matched against tracks, so
// they are refused at the boundary; callers should skip such features
// upstream.
var ErrNilGeometry = errors.New("streetlights: nil geometry")

// ErrNonPointGeometry is returned by [Insert] for geometries that are not a
// single point. Every streetlight row stores exactly one H3 cell, so only
// single points can be recorded; callers should drop such features upstream.
var ErrNonPointGeometry = errors.New("streetlights: non-point geometry")

// StreetlightInsert is the per-source data needed to record a streetlight.
// It is the shared shape produced by each source-specific downloader
// (e.g. KTZH, Stadt ZH) and consumed by [Insert].
type StreetlightInsert struct {
	// SourceID is the identifier from the upstream data source.
	SourceID string

	// InsertedBy is the job kind that produced this row.
	// Used to scope deletes during refresh cycles.
	InsertedBy string

	// Geometry is the lamp location in WGS84. Must be a single point;
	// [Insert] returns [ErrNilGeometry] or [ErrNonPointGeometry] otherwise.
	Geometry *geojson.Geometry

	// Attribution is the data source credit shown to end users.
	Attribution attribute.Attribution
}

// Insert writes one streetlight row. The geometry is validated before any
// writes, so a rejected feature leaves no partial rows behind. The
// attribution is deduplicated into its own table.
//
// Returns [ErrNilGeometry] or [ErrNonPointGeometry] if the geometry is not a
// single point; otherwise returns any error from marshalling the geometry or
// from the underlying DB inserts.
func Insert(ctx context.Context, tx *db.Queries, s StreetlightInsert, now time.Time) error {
	pt, err := point(s.Geometry)
	if err != nil {
		return err
	}

	attributionID, err := tx.UpsertStreetlightAttribution(ctx, db.UpsertStreetlightAttributionParams{
		Attribution:     s.Attribution.Author,
		AttributionHref: s.Attribution.Source,
	})
	if err != nil {
		return fmt.Errorf("upsert attribution: %w", err)
	}

	cell, err := h3.LatLngToCell(h3.LatLng{Lat: pt.Lat(), Lng: pt.Lon()}, CellResolution)
	if err != nil {
		return fmt.Errorf("compute cell: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate uuid: %w", err)
	}

	geomJSON, err := json.Marshal(s.Geometry)
	if err != nil {
		return fmt.Errorf("marshal geometry: %w", err)
	}

	err = tx.InsertStreetlight(ctx, db.InsertStreetlightParams{
		Uuid:          id.String(),
		SourceID:      s.SourceID,
		InsertedBy:    s.InsertedBy,
		CreatedAt:     now,
		AttributionID: attributionID,
		Cell:          int64(cell),
		Geometry:      string(geomJSON),
	})
	if err != nil {
		return fmt.Errorf("insert streetlight: %w", err)
	}

	return nil
}

// point extracts the coordinate of a streetlight geometry, rejecting nil and
// non-point geometries.
func point(geom *geojson.Geometry) (orb.Point, error) {
	if geom == nil {
		return orb.Point{}, ErrNilGeometry
	}
	switch g := geom.Geometry().(type) {
	case orb.Point:
		return g, nil
	default:
		return orb.Point{}, fmt.Errorf("%w: %T", ErrNonPointGeometry, g)
	}
}
