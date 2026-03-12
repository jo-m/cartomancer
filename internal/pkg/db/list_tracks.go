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

	// OnlyOwnedByUser restricts results to tracks owned by UserID, ignoring public
	// visibility from other users. Has no effect when UserID is empty.
	OnlyOwnedByUser bool

	// OnlyStarred restricts results to tracks starred by ViewerUserID.
	// Has no effect when ViewerUserID is empty.
	OnlyStarred bool

	// ViewerUserID is the UUID of the user viewing the results, used to compute
	// the Starred field on each returned track. Empty string = anonymous viewer
	// (Starred is always false).
	ViewerUserID string

	// Public filters by visibility. nil = no filter.
	Public *bool

	// Enum multi-value filters. nil/empty = no filter (all values allowed).
	FileFormats []int64
	TrackTypes  []int64
	Sports      []int64
	SubSports   []int64

	// Tags filters by tag names. Empty = no filter.
	// When TagsAnd is true all listed tags must be present; otherwise any one suffices.
	Tags    []string
	TagsAnd bool

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

// TrackWithStarred pairs a Track with the viewing user's star status for that track.
type TrackWithStarred struct {
	Track
	Starred bool
}

// ListTracksResult holds a page of tracks and the total count.
type ListTracksResult struct {
	Tracks     []TrackWithStarred
	TotalCount int
}

// trackAllCols lists all 31 track columns with the tracks. table prefix,
// suitable for use in queries that JOIN against other tables.
const trackAllCols = `tracks.uuid, tracks.created_at, tracks.updated_at, tracks.initial_editing_completed, tracks.user_id, tracks.public, tracks.blob_id, tracks.file_format, tracks.original_filename, tracks.name, tracks.description, tracks.source, tracks.author, tracks.author_link_url, tracks.track_type, tracks.link_url, tracks.sport, tracks.sub_sport, tracks.total_distance_m, tracks.total_ascent_m, tracks.start_lat, tracks.start_lon, tracks.end_lat, tracks.end_lon, tracks.bounds_min_lat, tracks.bounds_min_lon, tracks.bounds_max_lat, tracks.bounds_max_lon, tracks.min_elevation_m, tracks.max_elevation_m, tracks.original_created_at`

// scanTrackWithStar scans a row containing all 31 track columns (in trackAllCols order)
// followed by a single integer starred column.
func scanTrackWithStar(rows *sql.Rows) (TrackWithStarred, error) {
	var i Track
	var starred int64
	err := rows.Scan(
		&i.Uuid,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.InitialEditingCompleted,
		&i.UserID,
		&i.Public,
		&i.BlobID,
		&i.FileFormat,
		&i.OriginalFilename,
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
		&i.BoundsMinLat,
		&i.BoundsMinLon,
		&i.BoundsMaxLat,
		&i.BoundsMaxLon,
		&i.MinElevationM,
		&i.MaxElevationM,
		&i.OriginalCreatedAt,
		&starred,
	)
	return TrackWithStarred{Track: i, Starred: starred != 0}, err
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
// Each returned track includes an Starred field indicating whether ViewerUserID
// has starred it (always false when ViewerUserID is empty).
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

	// Visibility filter.
	if p.UserID != "" && p.OnlyOwnedByUser {
		b.add("tracks.user_id = ?", p.UserID)
	} else if p.UserID != "" {
		b.add("(tracks.public = 1 OR tracks.user_id = ?)", p.UserID)
	} else {
		b.add("tracks.public = 1")
	}

	if p.Public != nil {
		if *p.Public {
			b.add("tracks.public = 1")
		} else {
			b.add("tracks.public = 0")
		}
	}

	// OnlyStarred uses the LEFT JOIN alias ts (joined on ViewerUserID).
	if p.OnlyStarred && p.ViewerUserID != "" {
		b.add("ts.track_id IS NOT NULL")
	}

	b.inInt64("tracks.file_format", p.FileFormats)
	b.inInt64("tracks.track_type", p.TrackTypes)
	b.inInt64("tracks.sport", p.Sports)
	b.inInt64("tracks.sub_sport", p.SubSports)

	if len(p.Tags) > 0 {
		placeholders := make([]string, len(p.Tags))
		args := make([]any, len(p.Tags))
		for i, tag := range p.Tags {
			placeholders[i] = "?"
			args[i] = tag
		}
		sub := "SELECT track_id FROM track_tags JOIN tags ON tags.id = track_tags.tag_id WHERE tags.tag IN (" + strings.Join(placeholders, ", ") + ")"
		if p.TagsAnd {
			sub += fmt.Sprintf(" GROUP BY track_id HAVING COUNT(DISTINCT tags.tag) = %d", len(p.Tags))
		}
		b.add("tracks.uuid IN ("+sub+")", args...)
	}

	if p.Name != nil {
		b.add("tracks.name LIKE ?", "%"+*p.Name+"%")
	}
	if p.Description != nil {
		b.add("tracks.description LIKE ?", "%"+*p.Description+"%")
	}
	if p.Source != nil {
		b.add("tracks.source LIKE ?", "%"+*p.Source+"%")
	}

	if p.CreatedAtMin != nil {
		b.add("tracks.created_at >= ?", *p.CreatedAtMin)
	}
	if p.CreatedAtMax != nil {
		b.add("tracks.created_at <= ?", *p.CreatedAtMax)
	}
	if p.UpdatedAtMin != nil {
		b.add("tracks.updated_at >= ?", *p.UpdatedAtMin)
	}
	if p.UpdatedAtMax != nil {
		b.add("tracks.updated_at <= ?", *p.UpdatedAtMax)
	}
	if p.OriginalCreatedAtMin != nil {
		b.add("tracks.original_created_at >= ?", *p.OriginalCreatedAtMin)
	}
	if p.OriginalCreatedAtMax != nil {
		b.add("tracks.original_created_at <= ?", *p.OriginalCreatedAtMax)
	}

	if p.TotalDistanceMMin != nil {
		b.add("tracks.total_distance_m >= ?", *p.TotalDistanceMMin)
	}
	if p.TotalDistanceMMax != nil {
		b.add("tracks.total_distance_m <= ?", *p.TotalDistanceMMax)
	}
	if p.TotalAscentMMin != nil {
		b.add("tracks.total_ascent_m >= ?", *p.TotalAscentMMin)
	}
	if p.TotalAscentMMax != nil {
		b.add("tracks.total_ascent_m <= ?", *p.TotalAscentMMax)
	}

	if p.StartLatMin != nil {
		b.add("tracks.start_lat >= ?", *p.StartLatMin)
	}
	if p.StartLatMax != nil {
		b.add("tracks.start_lat <= ?", *p.StartLatMax)
	}
	if p.StartLonMin != nil {
		b.add("tracks.start_lon >= ?", *p.StartLonMin)
	}
	if p.StartLonMax != nil {
		b.add("tracks.start_lon <= ?", *p.StartLonMax)
	}
	if p.EndLatMin != nil {
		b.add("tracks.end_lat >= ?", *p.EndLatMin)
	}
	if p.EndLatMax != nil {
		b.add("tracks.end_lat <= ?", *p.EndLatMax)
	}
	if p.EndLonMin != nil {
		b.add("tracks.end_lon >= ?", *p.EndLonMin)
	}
	if p.EndLonMax != nil {
		b.add("tracks.end_lon <= ?", *p.EndLonMax)
	}

	where := b.whereClause()
	// The LEFT JOIN computes starred for the viewer; an empty ViewerUserID never
	// matches any row, so starred is always 0 for anonymous callers.
	const join = " LEFT JOIN track_stars ts ON ts.track_id = tracks.uuid AND ts.user_id = ?"

	// The ViewerUserID arg comes first, before the WHERE args, matching the JOIN position.
	baseArgs := append([]any{p.ViewerUserID}, b.args...)

	var total int
	countSQL := "SELECT COUNT(*) FROM tracks" + join + where
	if err := d.ro.QueryRowContext(ctx, countSQL, baseArgs...).Scan(&total); err != nil {
		return ListTracksResult{}, fmt.Errorf("count tracks: %w", err)
	}

	offset := (p.Page - 1) * p.PageSize
	dataSQL := fmt.Sprintf(
		"SELECT %s, CASE WHEN ts.track_id IS NOT NULL THEN 1 ELSE 0 END AS starred FROM tracks%s%s ORDER BY tracks.created_at DESC LIMIT ? OFFSET ?",
		trackAllCols, join, where,
	)
	dataArgs := append(append([]any{}, baseArgs...), int64(p.PageSize), int64(offset))

	rows, err := d.ro.QueryContext(ctx, dataSQL, dataArgs...)
	if err != nil {
		return ListTracksResult{}, fmt.Errorf("list tracks: %w", err)
	}
	defer rows.Close()

	tracks := make([]TrackWithStarred, 0, p.PageSize)
	for rows.Next() {
		t, err := scanTrackWithStar(rows)
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
