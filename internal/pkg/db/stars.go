package db

import (
	"context"
	"database/sql"
	"fmt"
)

// GetTrackByUUIDForViewer returns the track with the given UUID together with
// a flag indicating whether viewerUserID has starred it. An empty viewerUserID
// (anonymous caller) always yields IsStarred = false. Returns sql.ErrNoRows
// when the track does not exist.
func (d *DB) GetTrackByUUIDForViewer(ctx context.Context, uuid, viewerUserID string) (TrackWithStarred, error) {
	query := fmt.Sprintf(
		"SELECT %s, CASE WHEN ts.track_id IS NOT NULL THEN 1 ELSE 0 END AS is_starred"+
			" FROM tracks LEFT JOIN track_stars ts ON ts.track_id = tracks.uuid AND ts.user_id = ?"+
			" WHERE tracks.uuid = ?",
		trackAllCols,
	)
	rows, err := d.ro.QueryContext(ctx, query, viewerUserID, uuid)
	if err != nil {
		return TrackWithStarred{}, fmt.Errorf("get track by uuid for viewer: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return TrackWithStarred{}, err
		}
		return TrackWithStarred{}, sql.ErrNoRows
	}
	return scanTrackWithStar(rows)
}

// ListTracksForEditingForViewer returns all tracks owned by userID that have not
// yet completed initial editing, together with that user's star status for each.
// Results are ordered by creation time, newest first.
func (d *DB) ListTracksForEditingForViewer(ctx context.Context, userID string) ([]TrackWithStarred, error) {
	query := fmt.Sprintf(
		"SELECT %s, CASE WHEN ts.track_id IS NOT NULL THEN 1 ELSE 0 END AS is_starred"+
			" FROM tracks LEFT JOIN track_stars ts ON ts.track_id = tracks.uuid AND ts.user_id = ?"+
			" WHERE tracks.user_id = ? AND tracks.initial_editing_completed = 0"+
			" ORDER BY tracks.created_at DESC",
		trackAllCols,
	)
	rows, err := d.ro.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("list tracks for editing for viewer: %w", err)
	}
	defer rows.Close()

	var tracks []TrackWithStarred
	for rows.Next() {
		t, err := scanTrackWithStar(rows)
		if err != nil {
			return nil, fmt.Errorf("scan track for editing: %w", err)
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

// GetStarredTracks returns the tracks starred by starredByUserID, filtered to
// those visible to viewerUserID (empty string = anonymous viewer), together with
// the IsStarred flag reflecting viewerUserID's own star status on each track.
// Results are ordered by star creation time, newest first.
func (d *DB) GetStarredTracks(ctx context.Context, starredByUserID, viewerUserID string) ([]TrackWithStarred, error) {
	var query string
	var args []any

	if viewerUserID == "" {
		query = fmt.Sprintf(
			"SELECT %s, 0 AS is_starred"+
				" FROM track_stars ts_owner"+
				" JOIN tracks ON tracks.uuid = ts_owner.track_id"+
				" WHERE ts_owner.user_id = ? AND tracks.public = 1"+
				" ORDER BY ts_owner.created_at DESC",
			trackAllCols,
		)
		args = []any{starredByUserID}
	} else {
		query = fmt.Sprintf(
			"SELECT %s, CASE WHEN ts_viewer.track_id IS NOT NULL THEN 1 ELSE 0 END AS is_starred"+
				" FROM track_stars ts_owner"+
				" JOIN tracks ON tracks.uuid = ts_owner.track_id"+
				" LEFT JOIN track_stars ts_viewer ON ts_viewer.track_id = tracks.uuid AND ts_viewer.user_id = ?"+
				" WHERE ts_owner.user_id = ? AND (tracks.public = 1 OR tracks.user_id = ?)"+
				" ORDER BY ts_owner.created_at DESC",
			trackAllCols,
		)
		args = []any{viewerUserID, starredByUserID, viewerUserID}
	}

	rows, err := d.ro.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get starred tracks: %w", err)
	}
	defer rows.Close()

	var tracks []TrackWithStarred
	for rows.Next() {
		t, err := scanTrackWithStar(rows)
		if err != nil {
			return nil, fmt.Errorf("scan starred track: %w", err)
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}
