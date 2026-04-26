package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/utl"
)

// TrackPolyline is a single row returned by [DB.ListTracksWithPolylines].
// It carries the small subset of track columns that the map view needs in
// addition to the encoded preview polyline.
type TrackPolyline struct {
	UUID           string
	UserID         string
	UserName       string
	Name           string
	TotalDistanceM float64
	TotalAscentM   float64
	BoundsMinLat   sql.NullFloat64
	BoundsMinLon   sql.NullFloat64
	BoundsMaxLat   sql.NullFloat64
	BoundsMaxLon   sql.NullFloat64
	Polyline       string
	Starred        bool
	UpdatedAt      time.Time
}

// ListTracksWithPolylinesResult holds a page of tracks-with-polylines and
// summary counts of the matched set.
type ListTracksWithPolylinesResult struct {
	// Tracks holds the rendered tracks (those with a non-NULL preview_polyline,
	// up to the requested limit).
	Tracks []TrackPolyline
	// TotalCount is the total number of tracks matching the filters
	// (regardless of preview_polyline state and limit).
	TotalCount int
	// PendingCount is the number of matching tracks whose preview_polyline
	// is still NULL (i.e. waiting on the backfill job).
	PendingCount int
	// MaxUpdatedAt is the most recent updated_at over the matched set, used
	// for ETag computation. Zero if the set is empty.
	MaxUpdatedAt time.Time
}

// ListTracksWithPolylines returns all tracks matching p (subject to limit)
// that have a non-NULL preview_polyline, alongside summary counts. The
// returned slice is ordered by the configured sort columns from p.
// Pagination on p is ignored; limit is applied after filtering.
func (d *DB) ListTracksWithPolylines(ctx context.Context, p ListTracksParams, limit int) (ListTracksWithPolylinesResult, error) {
	if limit < 1 {
		limit = 1
	}

	// Validate and default sort parameters.
	allowedSortCols := map[string]string{
		"created_at":       "tracks.created_at",
		"total_distance_m": "tracks.total_distance_m",
		"total_ascent_m":   "tracks.total_ascent_m",
	}
	sortCol, ok := allowedSortCols[p.SortBy]
	if !ok {
		sortCol = "tracks.created_at"
	}
	sortDir := "DESC"
	if p.SortOrder == "asc" {
		sortDir = "ASC"
	}

	// Convert radial filters to bounding boxes.
	if p.StartNearLat != nil && p.StartNearLon != nil && p.StartNearRadiusM != nil {
		dLat := latDeltaDeg(*p.StartNearRadiusM)
		dLon := lonDeltaDeg(*p.StartNearRadiusM, *p.StartNearLat)
		p.StartLatMin = utl.Ptr(*p.StartNearLat - dLat)
		p.StartLatMax = utl.Ptr(*p.StartNearLat + dLat)
		p.StartLonMin = utl.Ptr(*p.StartNearLon - dLon)
		p.StartLonMax = utl.Ptr(*p.StartNearLon + dLon)
	}
	if p.EndNearLat != nil && p.EndNearLon != nil && p.EndNearRadiusM != nil {
		dLat := latDeltaDeg(*p.EndNearRadiusM)
		dLon := lonDeltaDeg(*p.EndNearRadiusM, *p.EndNearLat)
		p.EndLatMin = utl.Ptr(*p.EndNearLat - dLat)
		p.EndLatMax = utl.Ptr(*p.EndNearLat + dLat)
		p.EndLonMin = utl.Ptr(*p.EndNearLon - dLon)
		p.EndLonMax = utl.Ptr(*p.EndNearLon + dLon)
	}

	b := buildTrackPredicate(p)
	where := b.whereClause()

	joins := " JOIN users ON users.uuid = tracks.user_id" +
		" LEFT JOIN track_stars ts ON ts.track_id = tracks.uuid AND ts.user_id = ?" +
		" LEFT JOIN track_geonames tg ON tg.track_id = tracks.uuid"

	baseArgs := append([]any{p.ViewerUserID}, b.args...)

	// Total count over the entire matched set (including not-yet-backfilled).
	var total int
	countSQL := "SELECT COUNT(*) FROM tracks" + joins + where
	if err := d.ro.QueryRowContext(ctx, countSQL, baseArgs...).Scan(&total); err != nil {
		return ListTracksWithPolylinesResult{}, fmt.Errorf("count tracks: %w", err)
	}

	// Pending count.
	var pending int
	pendingWhere := where
	if pendingWhere == "" {
		pendingWhere = " WHERE tracks.preview_polyline IS NULL"
	} else {
		pendingWhere += " AND tracks.preview_polyline IS NULL"
	}
	pendingSQL := "SELECT COUNT(*) FROM tracks" + joins + pendingWhere
	if err := d.ro.QueryRowContext(ctx, pendingSQL, baseArgs...).Scan(&pending); err != nil {
		return ListTracksWithPolylinesResult{}, fmt.Errorf("count pending tracks: %w", err)
	}

	// Max updated_at over the matched set, for ETag.
	var maxUpdatedRaw sql.NullTime
	maxSQL := "SELECT MAX(tracks.updated_at) FROM tracks" + joins + where
	if err := d.ro.QueryRowContext(ctx, maxSQL, baseArgs...).Scan(&maxUpdatedRaw); err != nil {
		return ListTracksWithPolylinesResult{}, fmt.Errorf("max updated_at: %w", err)
	}

	// Data query: only rows with a non-NULL preview_polyline.
	dataWhere := where
	if dataWhere == "" {
		dataWhere = " WHERE tracks.preview_polyline IS NOT NULL"
	} else {
		dataWhere += " AND tracks.preview_polyline IS NOT NULL"
	}
	dataSQL := fmt.Sprintf( // #nosec G201
		"SELECT tracks.uuid, tracks.user_id, users.name AS user_name, tracks.name, "+
			"tracks.total_distance_m, tracks.total_ascent_m, "+
			"tracks.bounds_min_lat, tracks.bounds_min_lon, tracks.bounds_max_lat, tracks.bounds_max_lon, "+
			"tracks.preview_polyline, "+
			"CASE WHEN ts.track_id IS NOT NULL THEN 1 ELSE 0 END AS starred, "+
			"tracks.updated_at "+
			"FROM tracks%s%s ORDER BY %s %s LIMIT ?",
		joins, dataWhere, sortCol, sortDir,
	)
	dataArgs := append(append([]any{}, baseArgs...), int64(limit))

	rows, err := d.ro.QueryContext(ctx, dataSQL, dataArgs...)
	if err != nil {
		return ListTracksWithPolylinesResult{}, fmt.Errorf("list track polylines: %w", err)
	}
	defer rows.Close()

	tracks := make([]TrackPolyline, 0, limit)
	for rows.Next() {
		var tp TrackPolyline
		var starred int64
		if err := rows.Scan(
			&tp.UUID,
			&tp.UserID,
			&tp.UserName,
			&tp.Name,
			&tp.TotalDistanceM,
			&tp.TotalAscentM,
			&tp.BoundsMinLat,
			&tp.BoundsMinLon,
			&tp.BoundsMaxLat,
			&tp.BoundsMaxLon,
			&tp.Polyline,
			&starred,
			&tp.UpdatedAt,
		); err != nil {
			return ListTracksWithPolylinesResult{}, fmt.Errorf("scan track polyline: %w", err)
		}
		tp.Starred = starred != 0
		tracks = append(tracks, tp)
	}
	if err := rows.Err(); err != nil {
		return ListTracksWithPolylinesResult{}, fmt.Errorf("iterate track polylines: %w", err)
	}

	res := ListTracksWithPolylinesResult{
		Tracks:       tracks,
		TotalCount:   total,
		PendingCount: pending,
	}
	if maxUpdatedRaw.Valid {
		res.MaxUpdatedAt = maxUpdatedRaw.Time
	}
	return res, nil
}
