package db

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/utl"
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

	// SortBy specifies the column to sort by. Valid values: "created_at",
	// "total_distance_m", "total_ascent_m". Defaults to "created_at".
	SortBy string
	// SortOrder specifies the sort direction. Valid values: "asc", "desc".
	// Defaults to "desc".
	SortOrder string

	// Page is 1-based. Defaults to 1.
	Page int
	// PageSize is the number of results per page. Defaults to 25.
	PageSize int
}

// TrackWithStarred pairs a Track with the viewing user's star status, the track user's info,
// and an optional forecast summary.
type TrackWithStarred struct {
	Track
	Starred        bool
	UserName       string
	UserAvatarSeed string
	GeonameLabel   string
	Forecast       TrackForecastSummary
}

// ListTracksResult holds a page of tracks and the total count.
type ListTracksResult struct {
	Tracks     []TrackWithStarred
	TotalCount int
}

// trackAllCols lists all 31 track columns followed by the owner's name, avatar_seed, and
// geoname label, with table prefixes for use in queries that JOIN against other tables.
const trackAllCols = `tracks.uuid, tracks.created_at, tracks.updated_at, tracks.initial_editing_completed, tracks.user_id, tracks.public, tracks.blob_id, tracks.file_format, tracks.original_filename, tracks.name, tracks.description, tracks.source, tracks.author, tracks.author_link_url, tracks.track_type, tracks.link_url, tracks.sport, tracks.sub_sport, tracks.total_distance_m, tracks.total_ascent_m, tracks.start_lat, tracks.start_lon, tracks.end_lat, tracks.end_lon, tracks.bounds_min_lat, tracks.bounds_min_lon, tracks.bounds_max_lat, tracks.bounds_max_lon, tracks.min_elevation_m, tracks.max_elevation_m, tracks.original_created_at, users.name AS user_name, users.avatar_seed AS user_avatar_seed, tg.label AS geoname_label`

// scanTrackWithStar scans a row containing all 31 track columns plus owner_name,
// owner_avatar_seed, and geoname_label (in trackAllCols order) followed by starred.
// The Forecast field is left at its zero value.
func scanTrackWithStar(rows *sql.Rows) (TrackWithStarred, error) {
	var i Track
	var starred int64
	var userName, userAvatarSeed string
	var geonameLabel sql.NullString
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
		&userName,
		&userAvatarSeed,
		&geonameLabel,
		&starred,
	)
	return TrackWithStarred{
		Track:          i,
		Starred:        starred != 0,
		UserName:       userName,
		UserAvatarSeed: userAvatarSeed,
		GeonameLabel:   geonameLabel.String,
	}, err
}

// scanTrackWithStarAndForecast scans the same columns as scanTrackWithStar plus
// the 8 forecast columns from a LEFT JOIN on track_forecasts.
func scanTrackWithStarAndForecast(rows *sql.Rows) (TrackWithStarred, error) {
	var i Track
	var starred int64
	var userName, userAvatarSeed string
	var geonameLabel sql.NullString
	var fc TrackForecastSummary
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
		&userName,
		&userAvatarSeed,
		&geonameLabel,
		&starred,
		&fc.ForecastReferenceTime,
		&fc.StartTime,
		&fc.AvgTemperatureC,
		&fc.TotalPrecipitationMm,
		&fc.WindHeadMs,
		&fc.WindRightMs,
		&fc.WindTailMs,
		&fc.WindLeftMs,
	)
	return TrackWithStarred{
		Track:          i,
		Starred:        starred != 0,
		UserName:       userName,
		UserAvatarSeed: userAvatarSeed,
		GeonameLabel:   geonameLabel.String,
		Forecast:       fc,
	}, err
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
		b.add("(tracks.name LIKE ? OR tg.label LIKE ?)", "%"+*p.Name+"%", "%"+*p.Name+"%")
	}
	if p.Description != nil {
		b.add("tracks.description LIKE ?", "%"+*p.Description+"%")
	}
	if p.Source != nil {
		b.add("tracks.source LIKE ?", "%"+*p.Source+"%")
	}

	if p.CreatedAtMin != nil {
		b.add("datetime(tracks.created_at) >= datetime(?)", *p.CreatedAtMin)
	}
	if p.CreatedAtMax != nil {
		b.add("datetime(tracks.created_at) <= datetime(?)", *p.CreatedAtMax)
	}
	if p.UpdatedAtMin != nil {
		b.add("datetime(tracks.updated_at) >= datetime(?)", *p.UpdatedAtMin)
	}
	if p.UpdatedAtMax != nil {
		b.add("datetime(tracks.updated_at) <= datetime(?)", *p.UpdatedAtMax)
	}
	if p.OriginalCreatedAtMin != nil {
		b.add("datetime(tracks.original_created_at) >= datetime(?)", *p.OriginalCreatedAtMin)
	}
	if p.OriginalCreatedAtMax != nil {
		b.add("datetime(tracks.original_created_at) <= datetime(?)", *p.OriginalCreatedAtMax)
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
	// The JOINs fetch owner info, starred status for the viewer, geoname label,
	// and the forecast summary (only included when start_time is still in the future).
	joins := " JOIN users ON users.uuid = tracks.user_id" +
		" LEFT JOIN track_stars ts ON ts.track_id = tracks.uuid AND ts.user_id = ?" +
		" LEFT JOIN track_geonames tg ON tg.track_id = tracks.uuid" +
		" LEFT JOIN track_forecasts tf ON tf.track_uuid = tracks.uuid AND datetime(tf.start_time) > datetime(?)"

	now := time.Now()

	// The ViewerUserID and now args come first, before the WHERE args, matching the JOIN positions.
	baseArgs := append([]any{p.ViewerUserID, now}, b.args...)

	var total int
	countSQL := "SELECT COUNT(*) FROM tracks" + joins + where
	if err := d.ro.QueryRowContext(ctx, countSQL, baseArgs...).Scan(&total); err != nil {
		return ListTracksResult{}, fmt.Errorf("count tracks: %w", err)
	}

	const forecastCols = ", tf.forecast_reference_time, tf.start_time, tf.avg_temperature_c, tf.total_precipitation_mm, tf.wind_head_ms, tf.wind_right_ms, tf.wind_tail_ms, tf.wind_left_ms"

	offset := (p.Page - 1) * p.PageSize
	dataSQL := fmt.Sprintf(
		"SELECT %s, CASE WHEN ts.track_id IS NOT NULL THEN 1 ELSE 0 END AS starred%s FROM tracks%s%s ORDER BY %s %s LIMIT ? OFFSET ?",
		trackAllCols, forecastCols, joins, where, sortCol, sortDir,
	)
	dataArgs := append(append([]any{}, baseArgs...), int64(p.PageSize), int64(offset))

	rows, err := d.ro.QueryContext(ctx, dataSQL, dataArgs...)
	if err != nil {
		return ListTracksResult{}, fmt.Errorf("list tracks: %w", err)
	}
	defer rows.Close()

	tracks := make([]TrackWithStarred, 0, p.PageSize)
	for rows.Next() {
		t, err := scanTrackWithStarAndForecast(rows)
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

// TrackForecastSummary holds the pre-computed forecast summary for a track.
type TrackForecastSummary struct {
	ForecastReferenceTime sql.NullTime
	StartTime             sql.NullTime
	AvgTemperatureC       sql.NullFloat64
	TotalPrecipitationMm  sql.NullFloat64
	WindHeadMs            sql.NullFloat64
	WindRightMs           sql.NullFloat64
	WindTailMs            sql.NullFloat64
	WindLeftMs            sql.NullFloat64
}

// HasData reports whether the summary contains any forecast data.
func (s TrackForecastSummary) HasData() bool {
	return s.ForecastReferenceTime.Valid
}
