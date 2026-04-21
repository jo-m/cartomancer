package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ErrBulkDeleteMismatch is returned when the number of rows affected
// does not match the number of UUIDs, meaning some tracks were not found
// or not owned by the user.
var ErrBulkDeleteMismatch = errors.New("bulk delete mismatch")

// BulkDeleteTracks deletes all given tracks in a single transaction.
// All tracks must belong to the given user; cascades handle related tables.
// Returns ErrBulkDeleteMismatch if any track was not found or not owned.
func (d *DB) BulkDeleteTracks(ctx context.Context, uuids []string, userID string) error {
	if len(uuids) == 0 {
		return nil
	}

	placeholders := make([]string, len(uuids))
	args := make([]any, len(uuids))
	for i, id := range uuids {
		placeholders[i] = "?"
		args[i] = id
	}
	args = append(args, userID)

	// Only static "?" placeholders are interpolated; all user values go through args.
	query := fmt.Sprintf( // #nosec G201
		"DELETE FROM tracks WHERE uuid IN (%s) AND user_id = ?",
		strings.Join(placeholders, ", "),
	)

	return d.WithRWTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("bulk delete tracks: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("rows affected: %w", err)
		}

		if int(affected) != len(uuids) {
			return ErrBulkDeleteMismatch
		}

		return nil
	})
}
