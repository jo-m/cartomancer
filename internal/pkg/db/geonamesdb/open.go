package geonamesdb

import (
	"context"
	"database/sql"
	"embed"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

const migrationsDir = "migrations"

//go:embed migrations/*.sql
var embedMigrations embed.FS

// DB wraps a [db.DB] and provides typed query access for geonames tables.
type DB struct {
	*db.DB
}

// geonamesPragmas overrides the default SQLite pragmas for the geonames
// database. The geonames DB is a bulk-import workload where the full dataset
// (~12M rows) is re-imported periodically. Conservative limits prevent OOM
// kills in memory-constrained containers.
var geonamesPragmas = map[string]string{
	"cache_size": "-64000",  // 64 MB (negative = KiB)
	"mmap_size":  "0",       // Disable mmap; import is sequential, not random-read.
	"temp_store": "DEFAULT", // Let SQLite use temp files for sorting during index/FTS builds.
}

// Open opens the geonames database at path and runs migrations.
//
// Parameters:
//   - ctx: context for logging and migration execution.
//   - path: filesystem path for the geonames SQLite database file.
func Open(ctx context.Context, path string) (*DB, error) {
	d, err := db.Open(ctx, path, embedMigrations, migrationsDir, geonamesPragmas)
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
