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

	query := fmt.Sprintf(
		"UPDATE tracks SET initial_editing_completed = 1 WHERE uuid IN (%s) AND user_id = ?",
		strings.Join(placeholders, ", "),
	)

	tx, err := d.rw.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

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

	return tx.Commit()
}
