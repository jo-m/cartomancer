package db

import (
	"context"
	"fmt"
	"strings"
)

// GetActiveRoadClosuresByCells returns all road closures that have at least one
// H3 res-7 cell in common with the given set of cells. Closures whose ends_at
// is in the past are excluded.
func (d *DB) GetActiveRoadClosuresByCells(ctx context.Context, cells []int64) ([]RoadClosure, error) {
	if len(cells) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(cells))
	args := make([]any, len(cells))
	for i, c := range cells {
		placeholders[i] = "?"
		args[i] = c
	}

	// Only static "?" placeholders are interpolated; all user values go through args.
	query := fmt.Sprintf( // #nosec G201
		"SELECT DISTINCT rc.uuid, rc.source_id, rc.inserted_by, rc.created_at,"+
			" rc.type, rc.starts_at, rc.ends_at, rc.title, rc.reason,"+
			" rc.description, rc.content_provider, rc.geometry,"+
			" rc.attribution, rc.attribution_href"+
			" FROM road_closures rc"+
			" JOIN road_closure_cells_res7 cc ON cc.road_closure_id = rc.uuid"+
			" WHERE cc.cell IN (%s)"+
			" AND (rc.ends_at IS NULL OR rc.ends_at >= datetime('now'))",
		strings.Join(placeholders, ", "),
	)

	rows, err := d.ro.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get road closures by cells: %w", err)
	}
	defer rows.Close()

	var closures []RoadClosure
	for rows.Next() {
		var rc RoadClosure
		err := rows.Scan(
			&rc.Uuid, &rc.SourceID, &rc.InsertedBy, &rc.CreatedAt,
			&rc.Type, &rc.StartsAt, &rc.EndsAt, &rc.Title, &rc.Reason,
			&rc.Description, &rc.ContentProvider, &rc.Geometry,
			&rc.Attribution, &rc.AttributionHref,
		)
		if err != nil {
			return nil, fmt.Errorf("scan road closure: %w", err)
		}
		closures = append(closures, rc)
	}
	return closures, rows.Err()
}
