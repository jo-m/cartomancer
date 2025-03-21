package jobs

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"goweb/internal/pkg/db"
)

// randomID uniquely identifies this process.
// We use a random number instead of os.Getpid() because PIDs can be recycled.
var randomID = genRandInt64()

func genRandInt64() int64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic("rand.Read() cannot return err")
	}
	// #nosec G115 This is fine.
	return int64(binary.BigEndian.Uint64(b[:]))
}

func ensureSingleInstance(ctx context.Context, d *db.DB) error {
	return d.WithTx(ctx, func(tx *db.Queries) error {
		err := tx.InsertJobRunnerPID(ctx, randomID)
		if err != nil {
			return fmt.Errorf("only one instance allowed: %w", err)
		}

		return tx.DeleteOtherJobRunnerPIDs(ctx, randomID)
	})
}
