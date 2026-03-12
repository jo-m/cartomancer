package db

import (
	"context"
	"fmt"
	"strings"
)

// GetStarredStatusForTracks returns a map of trackUUID → true for each track in
// trackIDs that is starred by userID. Tracks not starred are absent from the map.
// Returns an empty map when userID is empty or trackIDs is empty.
func (d *DB) GetStarredStatusForTracks(ctx context.Context, userID string, trackIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(trackIDs))
	if userID == "" || len(trackIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(trackIDs))
	args := make([]any, 0, len(trackIDs)+1)
	args = append(args, userID)
	for i, id := range trackIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(
		"SELECT track_id FROM track_stars WHERE user_id = ? AND track_id IN (%s)",
		strings.Join(placeholders, ", "),
	)

	rows, err := d.ro.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get starred status for tracks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var trackID string
		if err := rows.Scan(&trackID); err != nil {
			return nil, fmt.Errorf("scan starred track id: %w", err)
		}
		result[trackID] = true
	}
	return result, rows.Err()
}

// GetStarredTracks returns the tracks starred by starredByUserID, filtered to
// those visible to viewerUserID (empty string = anonymous viewer).
// Results are ordered by star creation time, newest first.
func (d *DB) GetStarredTracks(ctx context.Context, starredByUserID, viewerUserID string) ([]Track, error) {
	if viewerUserID == "" {
		return d.QueryRO().GetStarredTracksPublicOnly(ctx, starredByUserID)
	}
	return d.QueryRO().GetStarredTracksVisible(ctx, GetStarredTracksVisibleParams{
		UserID:   starredByUserID,
		UserID_2: viewerUserID,
	})
}
