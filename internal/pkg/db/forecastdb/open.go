package forecastdb

import (
	"context"
	"database/sql"
	"embed"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

const migrationsDir = "migrations"

//go:embed migrations/*.sql
var embedMigrations embed.FS

// DB wraps a [db.DB] and provides typed query access for forecast tables.
type DB struct {
	*db.DB
}

// forecastPragmas overrides the default SQLite pragmas for the forecast database.
//
// The forecast DB stores GRIB2 files as BLOBs up to ~100 MiB each and may grow
// to several GiB total, while the whole app runs in ~2 GiB of RAM. Blob reads
// are one-shot (load, parse, discard) so caching blob pages provides no reuse
// and would only evict the small metadata working set; all other queries
// operate on metadata only. The overrides therefore aim to keep SQLite's
// in-process and OS-cache footprint small and predictable.
var forecastPragmas = map[string]string{
	// 2 MiB page cache. Enough for metadata indexes; small enough that one
	// 100 MiB blob read cannot evict useful pages or balloon RSS.
	"cache_size": "-2000",
	// Disable memory-mapped I/O. A 2 GiB mmap window over a multi-GiB DB
	// would compete with the process heap for resident pages on a 2 GiB host.
	"mmap_size": "0",
	// Spill temp tables and sort scratch to disk rather than RAM.
	"temp_store": "FILE",
	// Cap the WAL file size after checkpoint at 64 MiB, so a 100 MiB blob
	// insert does not leave a permanently bloated WAL on disk.
	"journal_size_limit": "67108864",
}

// Open opens the forecast database at path and runs migrations.
//
// Parameters:
//   - ctx: context for logging and migration execution.
//   - path: filesystem path for the forecast SQLite database file.
func Open(ctx context.Context, path string) (*DB, error) {
	d, err := db.Open(ctx, path, embedMigrations, migrationsDir, forecastPragmas)
	if err != nil {
		return nil, err
	}
	return &DB{DB: d}, nil
}

// QueryRO returns a [Queries] object with a read-only connection.
func (d *DB) QueryRO() *Queries {
	return New(d.RO())
}

// QueryRW returns a [Queries] object with a read/write connection.
func (d *DB) QueryRW() *Queries {
	return New(d.RW())
}

// WithTx runs the given function in a read/write transaction.
// If the function returns an error, the transaction is rolled back.
func (d *DB) WithTx(ctx context.Context, fn func(q *Queries) error) error {
	return d.WithRWTx(ctx, func(tx *sql.Tx) error {
		return fn(New(tx))
	})
}
