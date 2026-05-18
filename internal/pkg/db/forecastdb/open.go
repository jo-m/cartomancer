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

// forecastPragmas overrides the default SQLite pragmas for the forecast
// database on both the RW and RO connections.
//
// The forecast DB stores GRIB2 files as BLOBs up to ~100 MiB each and may grow
// to several GiB total, while the whole app runs in ~2 GiB of RAM. Blob reads
// are one-shot (load, parse, discard) so caching blob pages provides no reuse
// and would only evict the small metadata working set; all other queries
// operate on metadata only. The overrides therefore aim to keep SQLite's
// in-process and OS-cache footprint small and predictable, and to keep
// large-blob writes and deletes cheap.
var forecastPragmas = map[string]string{
	// 2 MiB page cache. Enough for metadata indexes; small enough that one
	// 100 MiB blob read cannot evict useful pages or balloon RSS.
	"cache_size": "-2000",
	// Disable memory-mapped I/O. A 2 GiB mmap window over a multi-GiB DB
	// would compete with the process heap for resident pages on a 2 GiB host.
	"mmap_size": "0",
	// Spill temp tables and sort scratch to disk rather than RAM.
	"temp_store": "FILE",
	// Cap the WAL file size after checkpoint at 256 MiB, matching the
	// autocheckpoint window so a single oversized insert does not leave the
	// WAL permanently bloated on disk.
	"journal_size_limit": "268435456",
}

// forecastPragmasRW carries pragmas that mutate the SQLite database header
// (page_size, auto_vacuum) or schedule writes during checkpointing
// (wal_autocheckpoint). They must only be applied to the read/write
// connection: the modernc.org/sqlite driver re-issues every _pragma= entry
// as a SET statement on each connection open, which fails on the read-only
// connection.
//
// page_size and auto_vacuum are persistent header settings: they take effect
// only when applied to a fresh database file. Changing them on an existing DB
// requires a full VACUUM. Since forecast data is regenerable, the deployment
// path is to rename/create a new DB file rather than VACUUM an existing one.
var forecastPragmasRW = map[string]string{
	// 32 KiB pages. A 100 MiB blob fits in ~3200 overflow pages instead of
	// ~25000 at the 4 KiB default, making blob reads, writes, and deletes
	// (free-list walks) roughly 8x cheaper in page-touch count.
	"page_size": "32768",
	// Incremental auto-vacuum so PRAGMA incremental_vacuum can return pages
	// to the OS after bulk deletes. Without this, the DB file never shrinks.
	"auto_vacuum": "INCREMENTAL",
	// Checkpoint after ~128 MiB of WAL (4000 * 32 KiB pages). Large enough
	// that one 100 MiB blob insert completes in a single WAL pass without
	// a mid-write checkpoint, small enough to bound crash recovery time.
	"wal_autocheckpoint": "4000",
}

// Open opens the forecast database at path and runs migrations.
//
// Parameters:
//   - ctx: context for logging and migration execution.
//   - path: filesystem path for the forecast SQLite database file.
func Open(ctx context.Context, path string) (*DB, error) {
	d, err := db.Open(ctx, path, embedMigrations, migrationsDir, forecastPragmas, forecastPragmasRW)
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
