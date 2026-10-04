package db

import (
	"context"
	"fmt"
	"strings"
)

// streetlightCellsChunkSize bounds the number of H3 cells per SQL statement.
// SQLite allows 32766 bound variables, but keeping the IN list small keeps
// statement preparation and the query plan cache happy.
const streetlightCellsChunkSize = 500

// StreetlightByCell is a streetlight row as needed by the track coverage
// computation: its H3 cell index, the GeoJSON point, and the source credit.
type StreetlightByCell struct {
	Cell            int64
	Geometry        string
	Attribution     string
	AttributionHref string
}

// GetStreetlightsByCells returns all streetlights whose H3 cell index is
// contained in cells. The input is deduplicated internally and queried in
// chunks so that arbitrarily long cell lists work. Returns nil for an empty
// input.
func (d *DB) GetStreetlightsByCells(ctx context.Context, cells []int64) ([]StreetlightByCell, error) {
	if len(cells) == 0 {
		return nil, nil
	}

	seen := make(map[int64]struct{}, len(cells))
	deduped := make([]int64, 0, len(cells))
	for _, c := range cells {
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		deduped = append(deduped, c)
	}

	var out []StreetlightByCell
	for start := 0; start < len(deduped); start += streetlightCellsChunkSize {
		chunk := deduped[start:min(start+streetlightCellsChunkSize, len(deduped))]

		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for i, c := range chunk {
			placeholders[i] = "?"
			args[i] = c
		}

		// Only static "?" placeholders are interpolated; all user values go through args.
		query := fmt.Sprintf( // #nosec G201
			"SELECT s.cell, s.geometry, a.attribution, a.attribution_href"+
				" FROM streetlights s"+
				" JOIN streetlight_attributions a ON a.id = s.attribution_id"+
				" WHERE s.cell IN (%s)",
			strings.Join(placeholders, ", "),
		)

		if err := d.queryStreetlightsByCells(ctx, query, args, &out); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// queryStreetlightsByCells runs one chunk query and appends the rows to out.
func (d *DB) queryStreetlightsByCells(ctx context.Context, query string, args []any, out *[]StreetlightByCell) error {
	rows, err := d.ro.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("get streetlights by cells: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s StreetlightByCell
		if err := rows.Scan(&s.Cell, &s.Geometry, &s.Attribution, &s.AttributionHref); err != nil {
			return fmt.Errorf("scan streetlight: %w", err)
		}
		*out = append(*out, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("get streetlights by cells: %w", err)
	}
	return nil
}
