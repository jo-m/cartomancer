package roadclosures

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb/geojson"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

// ErrNilGeometry is returned by [Insert] when called with a nil geometry.
// Closures without a geometry can never be matched against tracks, so they
// are refused at the boundary; callers should skip such features upstream.
var ErrNilGeometry = errors.New("roadclosures: nil geometry")

// ClosureInsert is the per-source data needed to record a road closure.
// It is the shared shape produced by each source-specific downloader
// (e.g. ASTRA, ZH) and consumed by [Insert].
type ClosureInsert struct {
	// SourceID is the identifier from the upstream data source.
	SourceID string

	// InsertedBy is the job kind that produced this row.
	// Used to scope deletes during refresh cycles.
	InsertedBy string

	// Type is the closure kind, normalised to [ClosedWay] or [Detour]
	// so that the API can be source-agnostic.
	Type ClosureType

	StartsAt sql.NullTime
	EndsAt   sql.NullTime

	Reason          sql.NullString
	Title           string
	Description     sql.NullString
	ContentProvider sql.NullString

	// Geometry is the closure footprint in WGS84. Must be non-nil;
	// [Insert] returns [ErrNilGeometry] otherwise.
	Geometry *geojson.Geometry

	// Attribution is the data source credit shown to end users.
	Attribution attribute.Attribution
}

// Insert writes one road closure row and its res-7 H3 cells. Caller supplies
// the transaction and the current time so that an entire refresh cycle shares
// the same created_at value.
//
// Returns [ErrNilGeometry] if c.Geometry is nil; the row is not written in
// that case. Otherwise returns any error from marshalling the geometry or
// from the underlying DB inserts.
func Insert(ctx context.Context, tx *db.Queries, c ClosureInsert, now time.Time) error {
	if c.Geometry == nil {
		return ErrNilGeometry
	}

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate uuid: %w", err)
	}

	geomJSON, err := json.Marshal(c.Geometry)
	if err != nil {
		return fmt.Errorf("marshal geometry: %w", err)
	}

	err = tx.InsertRoadClosure(ctx, db.InsertRoadClosureParams{
		Uuid:            id.String(),
		SourceID:        c.SourceID,
		InsertedBy:      c.InsertedBy,
		CreatedAt:       now,
		Type:            int64(c.Type),
		StartsAt:        c.StartsAt,
		EndsAt:          c.EndsAt,
		Reason:          c.Reason,
		Title:           c.Title,
		Description:     c.Description,
		ContentProvider: c.ContentProvider,
		Geometry:        string(geomJSON),
		Attribution:     c.Attribution.Author,
		AttributionHref: c.Attribution.Source,
	})
	if err != nil {
		return fmt.Errorf("insert road closure: %w", err)
	}

	cells := geometryCells(c.Geometry.Geometry(), CellResolution)
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

// NullString returns a sql.NullString that is valid only when s is non-empty.
func NullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
