package roadclosures

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb/geojson"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

// ClosureInsert is the per-source data needed to record a road closure.
// It is the shared shape produced by each source-specific downloader
// (e.g. astra, zh) and consumed by [Insert].
type ClosureInsert struct {
	// SourceID is the identifier from the upstream data source.
	SourceID string

	// InsertedBy is the job kind that produced this row.
	// Used to scope deletes during refresh cycles.
	InsertedBy string

	// Type is the closure kind, normalised to one of "detour" or
	// "closed_way" so that the API can be source-agnostic.
	Type string

	StartsAt sql.NullTime
	EndsAt   sql.NullTime

	Reason          sql.NullString
	Title           string
	Description     sql.NullString
	ContentProvider sql.NullString

	// Geometry is the closure footprint in WGS84. May be nil, in which case
	// no H3 cells are stored and the closure cannot be matched against tracks.
	Geometry *geojson.Geometry

	// Attribution is the data source credit shown to end users.
	Attribution attribute.Attribution
}

// Insert writes one road closure row and its res-7 H3 cells. Caller supplies
// the transaction and the current time so that an entire refresh cycle shares
// the same created_at value.
//
// Returns an error if the geometry cannot be marshalled or any DB insert fails.
// When c.Geometry is nil, the cell inserts are skipped but the row is still written.
func Insert(ctx context.Context, tx *db.Queries, c ClosureInsert, now time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate uuid: %w", err)
	}

	var geomJSON []byte
	if c.Geometry != nil {
		geomJSON, err = json.Marshal(c.Geometry)
		if err != nil {
			return fmt.Errorf("marshal geometry: %w", err)
		}
	}

	err = tx.InsertRoadClosure(ctx, db.InsertRoadClosureParams{
		Uuid:            id.String(),
		SourceID:        c.SourceID,
		InsertedBy:      c.InsertedBy,
		CreatedAt:       now,
		Type:            c.Type,
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

	if c.Geometry == nil {
		return nil
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
