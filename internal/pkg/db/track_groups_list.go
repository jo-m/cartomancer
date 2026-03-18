package db

import (
	"context"
	"fmt"
	"strings"
)

// GetTracksByUUIDs returns tracks matching the given UUIDs, with star status for the viewer.
// Only tracks visible to the viewer are returned (public, or owned by the viewer).
func (d *DB) GetTracksByUUIDs(ctx context.Context, uuids []string, viewerUserID string) ([]TrackWithStarred, error) {
	if len(uuids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(uuids))
	args := make([]any, 0, len(uuids)+2)
	for i, id := range uuids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	inClause := strings.Join(placeholders, ", ")

	var query string
	if viewerUserID == "" {
		query = fmt.Sprintf(
			"SELECT %s, 0 AS starred"+
				" FROM tracks"+
				" JOIN users ON users.uuid = tracks.user_id"+
				" LEFT JOIN track_geonames tg ON tg.track_id = tracks.uuid"+
				" WHERE tracks.uuid IN (%s) AND tracks.public = 1"+
				" ORDER BY tracks.name",
			trackAllCols, inClause,
		)
	} else {
		query = fmt.Sprintf(
			"SELECT %s, CASE WHEN ts.track_id IS NOT NULL THEN 1 ELSE 0 END AS starred"+
				" FROM tracks"+
				" JOIN users ON users.uuid = tracks.user_id"+
				" LEFT JOIN track_stars ts ON ts.track_id = tracks.uuid AND ts.user_id = ?"+
				" LEFT JOIN track_geonames tg ON tg.track_id = tracks.uuid"+
				" WHERE tracks.uuid IN (%s) AND (tracks.public = 1 OR tracks.user_id = ?)"+
				" ORDER BY tracks.name",
			trackAllCols, inClause,
		)
		args = append([]any{viewerUserID}, args...)
		args = append(args, viewerUserID)
	}

	rows, err := d.ro.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get tracks by uuids: %w", err)
	}
	defer rows.Close()

	var tracks []TrackWithStarred
	for rows.Next() {
		t, err := scanTrackWithStar(rows)
		if err != nil {
			return nil, fmt.Errorf("scan track by uuid: %w", err)
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}
