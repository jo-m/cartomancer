package streetlights

import (
	"bytes"
	"context"
	"database/sql"
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

// ErrNonPointGeometry is returned by [Insert] for non-point geometries.
// Every streetlight data source models lamps as points.
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

	// ContentProvider names the organization that operates the lights.
	ContentProvider sql.NullString

	// Properties holds the source's feature attributes verbatim as a JSON
	// object. Empty, missing and JSON null values are stored as an empty
	// object.
	Properties json.RawMessage

	// Geometry is the lamp location in WGS84. Must be a point or multi-point;
	// [Insert] returns [ErrNilGeometry] or [ErrNonPointGeometry] otherwise.
	Geometry *geojson.Geometry

	// Attribution is the data source credit shown to end users.
	Attribution attribute.Attribution
}

// Insert writes one streetlight row and its res-9 H3 cells. Caller supplies
// the transaction and the current time so that an entire refresh cycle shares
// the same created_at value.
//
// Returns [ErrNilGeometry] or [ErrNonPointGeometry] if the geometry is not a
// point; the row is not written in those cases. Otherwise returns any error
// from marshalling the geometry or from the underlying DB inserts.
func Insert(ctx context.Context, tx *db.Queries, s StreetlightInsert, now time.Time) error {
	pts, err := points(s.Geometry)
	if err != nil {
		return err
	}

	props, err := propertiesJSON(s.Properties)
	if err != nil {
		return err
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
		Uuid:            id.String(),
		SourceID:        s.SourceID,
		InsertedBy:      s.InsertedBy,
		CreatedAt:       now,
		ContentProvider: s.ContentProvider,
		Properties:      props,
		Geometry:        string(geomJSON),
		Attribution:     s.Attribution.Author,
		AttributionHref: s.Attribution.Source,
	})
	if err != nil {
		return fmt.Errorf("insert streetlight: %w", err)
	}

	for _, pt := range pts {
		cell, err := h3.LatLngToCell(h3.LatLng{Lat: pt.Lat(), Lng: pt.Lon()}, CellResolution)
		if err != nil {
			return fmt.Errorf("compute cell: %w", err)
		}
		err = tx.InsertStreetlightCellRes9(ctx, db.InsertStreetlightCellRes9Params{
			StreetlightID: id.String(),
			Cell:          int64(cell),
		})
		if err != nil {
			return fmt.Errorf("insert cell: %w", err)
		}
	}

	return nil
}

// points extracts the point coordinates of a streetlight geometry, rejecting
// nil and non-point geometries.
func points(geom *geojson.Geometry) ([]orb.Point, error) {
	if geom == nil {
		return nil, ErrNilGeometry
	}
	switch g := geom.Geometry().(type) {
	case orb.Point:
		return []orb.Point{g}, nil
	case orb.MultiPoint:
		return []orb.Point(g), nil
	default:
		return nil, fmt.Errorf("%w: %T", ErrNonPointGeometry, g)
	}
}

// propertiesJSON normalises raw properties for storage. Empty, missing and
// JSON null values become an empty object; anything that is not a JSON object
// is rejected.
func propertiesJSON(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "{}", nil
	}
	if trimmed[0] != '{' || !json.Valid(trimmed) {
		return "", fmt.Errorf("streetlights: properties is not a valid JSON object")
	}
	return string(trimmed), nil
}

// NullString returns a sql.NullString that is valid only when s is non-empty.
func NullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
