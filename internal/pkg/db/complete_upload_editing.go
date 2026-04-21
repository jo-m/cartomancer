package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// CompleteEditing sets initial_editing_completed = 1 for the given tracks.
// All tracks must belong to the given user.
// Returns ErrBulkUpdateMismatch if any track was not found or not owned by the user.
func (d *DB) CompleteEditing(ctx context.Context, userID string, uuids []string) error {
	if len(uuids) == 0 {
		return nil
	}

	placeholders := make([]string, len(uuids))
	args := make([]any, 0, len(uuids)+1)
	for i, id := range uuids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	args = append(args, userID)

	// Only static "?" placeholders are interpolated; all user values go through args.
	query := fmt.Sprintf( // #nosec G201
		"UPDATE tracks SET initial_editing_completed = 1 WHERE uuid IN (%s) AND user_id = ?",
		strings.Join(placeholders, ", "),
	)

	return d.WithRWTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("complete editing: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("rows affected: %w", err)
		}

		if int(affected) != len(uuids) {
			return ErrBulkUpdateMismatch
		}

		return nil
	})
}
