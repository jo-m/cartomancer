package db

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

// ListTracksParams defines the filter and pagination parameters for listing tracks.
type ListTracksParams struct {
	// UserID is the current user's UUID. If empty, only public tracks are returned.
	UserID string

	// Public filters by visibility. nil = no filter.
	Public *bool

	// Enum multi-value filters. nil/empty = no filter (all values allowed).
	FileFormats []int64
	TrackTypes  []int64
	Sports      []int64
	SubSports   []int64

	// Text LIKE filters (substring match). nil = no filter.
	Name        *string
	Description *string
	Source      *string

	// Datetime range filters. nil = no filter.
	CreatedAtMin         *time.Time
	CreatedAtMax         *time.Time
	UpdatedAtMin         *time.Time
	UpdatedAtMax         *time.Time
	OriginalCreatedAtMin *time.Time
	OriginalCreatedAtMax *time.Time

	// Numeric range filters. nil = no filter.
	TotalDistanceMMin *float64
	TotalDistanceMMax *float64
	TotalAscentMMin   *float64
	TotalAscentMMax   *float64

	// Coordinate bounding box filters. nil = no filter.
	StartLatMin *float64
	StartLatMax *float64
	StartLonMin *float64
	StartLonMax *float64
	EndLatMin   *float64
	EndLatMax   *float64
	EndLonMin   *float64
	EndLonMax   *float64

	// Radial filter for start coordinates.
	// All three must be non-nil for the filter to apply.
	// Converted to a bounding box internally.
	StartNearLat     *float64
	StartNearLon     *float64
	StartNearRadiusM *float64

	// Radial filter for end coordinates.
	// All three must be non-nil for the filter to apply.
	// Converted to a bounding box internally.
	EndNearLat     *float64
	EndNearLon     *float64
	EndNearRadiusM *float64

	// Page is 1-based. Defaults to 1.
	Page int
	// PageSize is the number of results per page. Defaults to 25.
	PageSize int
}

// ListTracksResult holds a page of tracks and the total count.
type ListTracksResult struct {
	Tracks     []Track
	TotalCount int
}

// trackScanCols must match the column order in the Track struct exactly.
const trackScanCols = `uuid, created_at, updated_at, user_id, public, blob_id, file_format, name, description, source, author, author_link_url, track_type, link_url, sport, sub_sport, total_distance_m, total_ascent_m, start_lat, start_lon, end_lat, end_lon, original_created_at`

func scanTrack(rows *sql.Rows) (Track, error) {
	var i Track
	err := rows.Scan(
		&i.Uuid,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.UserID,
		&i.Public,
		&i.BlobID,
		&i.FileFormat,
		&i.Name,
		&i.Description,
		&i.Source,
		&i.Author,
		&i.AuthorLinkUrl,
		&i.TrackType,
		&i.LinkUrl,
		&i.Sport,
		&i.SubSport,
		&i.TotalDistanceM,
		&i.TotalAscentM,
		&i.StartLat,
		&i.StartLon,
		&i.EndLat,
		&i.EndLon,
		&i.OriginalCreatedAt,
	)
	return i, err
}

type queryBuilder struct {
	where []string
	args  []any
}

func (b *queryBuilder) add(cond string, args ...any) {
	b.where = append(b.where, cond)
	b.args = append(b.args, args...)
}

func (b *queryBuilder) inInt64(col string, vals []int64) {
	if len(vals) == 0 {
		return
	}
	placeholders := make([]string, len(vals))
	args := make([]any, len(vals))
	for i, v := range vals {
		placeholders[i] = "?"
		args[i] = v
	}
	b.add(col+" IN ("+strings.Join(placeholders, ", ")+")", args...)
}

func (b *queryBuilder) whereClause() string {
	if len(b.where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(b.where, " AND ")
}

// latDeltaDeg converts a radius in meters to a latitude delta in degrees.
func latDeltaDeg(radiusM float64) float64 {
	return radiusM / 111320.0
}

// lonDeltaDeg converts a radius in meters to a longitude delta in degrees at the given latitude.
func lonDeltaDeg(radiusM, lat float64) float64 {
	cosLat := math.Cos(lat * math.Pi / 180)
	if math.Abs(cosLat) < 1e-10 {
		return 180.0
	}
	return radiusM / (111320.0 * math.Abs(cosLat))
}

func ptr[T any](v T) *T { return &v }

// ListTracks returns a paginated list of tracks matching the given filters.
func (d *DB) ListTracks(ctx context.Context, p ListTracksParams) (ListTracksResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 25
	}

	// Convert radial filters to bounding boxes.
	if p.StartNearLat != nil && p.StartNearLon != nil && p.StartNearRadiusM != nil {
		dLat := latDeltaDeg(*p.StartNearRadiusM)
		dLon := lonDeltaDeg(*p.StartNearRadiusM, *p.StartNearLat)
		p.StartLatMin = ptr(*p.StartNearLat - dLat)
		p.StartLatMax = ptr(*p.StartNearLat + dLat)
		p.StartLonMin = ptr(*p.StartNearLon - dLon)
		p.StartLonMax = ptr(*p.StartNearLon + dLon)
	}
	if p.EndNearLat != nil && p.EndNearLon != nil && p.EndNearRadiusM != nil {
		dLat := latDeltaDeg(*p.EndNearRadiusM)
		dLon := lonDeltaDeg(*p.EndNearRadiusM, *p.EndNearLat)
		p.EndLatMin = ptr(*p.EndNearLat - dLat)
		p.EndLatMax = ptr(*p.EndNearLat + dLat)
		p.EndLonMin = ptr(*p.EndNearLon - dLon)
		p.EndLonMax = ptr(*p.EndNearLon + dLon)
	}

	var b queryBuilder

	// Visibility: public OR owned by the current user.
	if p.UserID != "" {
		b.add("(public = 1 OR user_id = ?)", p.UserID)
	} else {
		b.add("public = 1")
	}

	if p.Public != nil {
		if *p.Public {
			b.add("public = 1")
		} else {
			b.add("public = 0")
		}
	}

	b.inInt64("file_format", p.FileFormats)
	b.inInt64("track_type", p.TrackTypes)
	b.inInt64("sport", p.Sports)
	b.inInt64("sub_sport", p.SubSports)

	if p.Name != nil {
		b.add("name LIKE ?", "%"+*p.Name+"%")
	}
	if p.Description != nil {
		b.add("description LIKE ?", "%"+*p.Description+"%")
	}
	if p.Source != nil {
		b.add("source LIKE ?", "%"+*p.Source+"%")
	}

	if p.CreatedAtMin != nil {
		b.add("created_at >= ?", *p.CreatedAtMin)
	}
	if p.CreatedAtMax != nil {
		b.add("created_at <= ?", *p.CreatedAtMax)
	}
	if p.UpdatedAtMin != nil {
		b.add("updated_at >= ?", *p.UpdatedAtMin)
	}
	if p.UpdatedAtMax != nil {
		b.add("updated_at <= ?", *p.UpdatedAtMax)
	}
	if p.OriginalCreatedAtMin != nil {
		b.add("original_created_at >= ?", *p.OriginalCreatedAtMin)
	}
	if p.OriginalCreatedAtMax != nil {
		b.add("original_created_at <= ?", *p.OriginalCreatedAtMax)
	}

	if p.TotalDistanceMMin != nil {
		b.add("total_distance_m >= ?", *p.TotalDistanceMMin)
	}
	if p.TotalDistanceMMax != nil {
		b.add("total_distance_m <= ?", *p.TotalDistanceMMax)
	}
	if p.TotalAscentMMin != nil {
		b.add("total_ascent_m >= ?", *p.TotalAscentMMin)
	}
	if p.TotalAscentMMax != nil {
		b.add("total_ascent_m <= ?", *p.TotalAscentMMax)
	}

	if p.StartLatMin != nil {
		b.add("start_lat >= ?", *p.StartLatMin)
	}
	if p.StartLatMax != nil {
		b.add("start_lat <= ?", *p.StartLatMax)
	}
	if p.StartLonMin != nil {
		b.add("start_lon >= ?", *p.StartLonMin)
	}
	if p.StartLonMax != nil {
		b.add("start_lon <= ?", *p.StartLonMax)
	}
	if p.EndLatMin != nil {
		b.add("end_lat >= ?", *p.EndLatMin)
	}
	if p.EndLatMax != nil {
		b.add("end_lat <= ?", *p.EndLatMax)
	}
	if p.EndLonMin != nil {
		b.add("end_lon >= ?", *p.EndLonMin)
	}
	if p.EndLonMax != nil {
		b.add("end_lon <= ?", *p.EndLonMax)
	}

	where := b.whereClause()

	var total int
	countSQL := "SELECT COUNT(*) FROM tracks" + where
	if err := d.ro.QueryRowContext(ctx, countSQL, b.args...).Scan(&total); err != nil {
		return ListTracksResult{}, fmt.Errorf("count tracks: %w", err)
	}

	offset := (p.Page - 1) * p.PageSize
	dataSQL := fmt.Sprintf(
		"SELECT %s FROM tracks%s ORDER BY created_at DESC LIMIT ? OFFSET ?",
		trackScanCols, where,
	)
	dataArgs := append(append([]any{}, b.args...), int64(p.PageSize), int64(offset))

	rows, err := d.ro.QueryContext(ctx, dataSQL, dataArgs...)
	if err != nil {
		return ListTracksResult{}, fmt.Errorf("list tracks: %w", err)
	}
	defer rows.Close()

	tracks := make([]Track, 0, p.PageSize)
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return ListTracksResult{}, fmt.Errorf("scan track: %w", err)
		}
		tracks = append(tracks, t)
	}
	if err := rows.Err(); err != nil {
		return ListTracksResult{}, fmt.Errorf("iterate tracks: %w", err)
	}

	return ListTracksResult{Tracks: tracks, TotalCount: total}, nil
}

// GetTagsForTracks returns a map of trackUUID → []tag for the given track UUIDs.
func (d *DB) GetTagsForTracks(ctx context.Context, trackUUIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(trackUUIDs))
	if len(trackUUIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(trackUUIDs))
	args := make([]any, len(trackUUIDs))
	for i, id := range trackUUIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		"SELECT tt.track_id, t.tag FROM tags t JOIN track_tags tt ON tt.tag_id = t.id WHERE tt.track_id IN (%s) ORDER BY t.tag",
		strings.Join(placeholders, ", "),
	)

	rows, err := d.ro.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get tags for tracks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var trackID, tag string
		if err := rows.Scan(&trackID, &tag); err != nil {
			return nil, fmt.Errorf("scan tag row: %w", err)
		}
		result[trackID] = append(result[trackID], tag)
	}
	return result, rows.Err()
}
