package db

import (
	"context"
	"database/sql"
	"fmt"
)

// TrackStatisticsParams defines the parameters for computing track statistics.
type TrackStatisticsParams struct {
	// UserID is the current user's UUID. If empty, only public tracks are included.
	UserID string
}

// TrackStatisticsResult holds aggregate stats for tracks visible to a user.
type TrackStatisticsResult struct {
	TotalDistanceMMin sql.NullFloat64
	TotalDistanceMMax sql.NullFloat64
	TotalAscentMMin   sql.NullFloat64
	TotalAscentMMax   sql.NullFloat64
}

// TrackStatistics returns aggregate statistics for tracks visible to the given user.
// All fields are NULL when there are no visible tracks.
func (d *DB) TrackStatistics(ctx context.Context, p TrackStatisticsParams) (TrackStatisticsResult, error) {
	var where string
	var args []any
	if p.UserID != "" {
		where = " WHERE (public = 1 OR user_id = ?)"
		args = []any{p.UserID}
	} else {
		where = " WHERE public = 1"
	}

	query := "SELECT MIN(total_distance_m), MAX(total_distance_m), MIN(total_ascent_m), MAX(total_ascent_m) FROM tracks" + where

	var r TrackStatisticsResult
	err := d.ro.QueryRowContext(ctx, query, args...).Scan(
		&r.TotalDistanceMMin,
		&r.TotalDistanceMMax,
		&r.TotalAscentMMin,
		&r.TotalAscentMMax,
	)
	if err != nil {
		return TrackStatisticsResult{}, fmt.Errorf("track statistics: %w", err)
	}

	return r, nil
}
