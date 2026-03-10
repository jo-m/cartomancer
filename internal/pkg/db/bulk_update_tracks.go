package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// BulkUpdateTracksParams defines the fields that can be bulk-updated.
// Only non-nil pointer fields are applied. Tags, if non-nil, replaces all tags on every track.
type BulkUpdateTracksParams struct {
	UUIDs         []string
	UserID        string
	Public        *bool
	Source        *string
	Author        *string
	AuthorLinkURL *string
	TrackType     *int64
	LinkURL       *string
	Sport         *int64
	SubSport      *int64
	Tags          *[]string
}

// ErrBulkUpdateMismatch is returned when the number of rows affected
// does not match the number of UUIDs, meaning some tracks were not found
// or not owned by the user.
var ErrBulkUpdateMismatch = errors.New("bulk update mismatch")

// BulkUpdateTracks updates the specified fields on all given tracks
// in a single transaction. All tracks must belong to the given user.
// Returns ErrBulkUpdateMismatch if any track was not found or not owned.
func (d *DB) BulkUpdateTracks(ctx context.Context, p BulkUpdateTracksParams) error {
	if len(p.UUIDs) == 0 {
		return nil
	}

	// Build dynamic SET clause.
	now := time.Now().UTC()
	setClauses := []string{"updated_at = ?"}
	args := []any{now}

	if p.Public != nil {
		var v int64
		if *p.Public {
			v = 1
		}
		setClauses = append(setClauses, "public = ?")
		args = append(args, v)
	}
	if p.Source != nil {
		setClauses = append(setClauses, "source = ?")
		args = append(args, toNullStr(*p.Source))
	}
	if p.Author != nil {
		setClauses = append(setClauses, "author = ?")
		args = append(args, toNullStr(*p.Author))
	}
	if p.AuthorLinkURL != nil {
		setClauses = append(setClauses, "author_link_url = ?")
		args = append(args, toNullStr(*p.AuthorLinkURL))
	}
	if p.TrackType != nil {
		setClauses = append(setClauses, "track_type = ?")
		args = append(args, *p.TrackType)
	}
	if p.LinkURL != nil {
		setClauses = append(setClauses, "link_url = ?")
		args = append(args, toNullStr(*p.LinkURL))
	}
	if p.Sport != nil {
		setClauses = append(setClauses, "sport = ?")
		args = append(args, *p.Sport)
	}
	if p.SubSport != nil {
		setClauses = append(setClauses, "sub_sport = ?")
		args = append(args, *p.SubSport)
	}

	// If only updated_at would change and there are no tags to replace, this is a no-op.
	if len(setClauses) == 1 && p.Tags == nil {
		return nil
	}

	// Build WHERE uuid IN (...) AND user_id = ?
	placeholders := make([]string, len(p.UUIDs))
	for i, id := range p.UUIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	args = append(args, p.UserID)

	query := fmt.Sprintf(
		"UPDATE tracks SET %s WHERE uuid IN (%s) AND user_id = ?",
		strings.Join(setClauses, ", "),
		strings.Join(placeholders, ", "),
	)

	tx, err := d.rw.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("bulk update tracks: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if int(affected) != len(p.UUIDs) {
		return ErrBulkUpdateMismatch
	}

	if p.Tags != nil {
		if err = bulkReplaceTags(ctx, tx, p.UUIDs, p.UserID, *p.Tags); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// bulkReplaceTags removes all existing tags from the given tracks and sets the new ones.
func bulkReplaceTags(ctx context.Context, tx *sql.Tx, uuids []string, userID string, tags []string) error {
	placeholders := make([]string, len(uuids))
	delArgs := make([]any, len(uuids))
	for i, id := range uuids {
		placeholders[i] = "?"
		delArgs[i] = id
	}

	_, err := tx.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM track_tags WHERE track_id IN (%s)", strings.Join(placeholders, ", ")),
		delArgs...,
	)
	if err != nil {
		return fmt.Errorf("delete track tags: %w", err)
	}

	for _, tag := range tags {
		var tagID int64
		err = tx.QueryRowContext(ctx,
			"INSERT INTO tags (tag, user_id) VALUES (?, ?) ON CONFLICT (tag, user_id) DO UPDATE SET tag = tag RETURNING id",
			tag, userID,
		).Scan(&tagID)
		if err != nil {
			return fmt.Errorf("upsert tag: %w", err)
		}
		for _, uuid := range uuids {
			_, err = tx.ExecContext(ctx,
				"INSERT INTO track_tags (track_id, tag_id) VALUES (?, ?)",
				uuid, tagID,
			)
			if err != nil {
				return fmt.Errorf("create track tag: %w", err)
			}
		}
	}

	return nil
}

func toNullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: s}
}
