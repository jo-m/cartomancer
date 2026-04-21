package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pressly/goose/v3"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	_ "modernc.org/sqlite" // Database driver.
)

const (
	driver  = "sqlite"
	dialect = "sqlite3"

	// MigrationsDir is the subdirectory name within the embedded FS that holds migration files.
	MigrationsDir = "migrations"
)

// EmbedMigrations contains the embedded SQL migration files for the main database.
//
//go:embed migrations/*.sql
var EmbedMigrations embed.FS

// defaultPragmas are the default SQLite pragmas applied to all databases.
// See https://phiresky.github.io/blog/2020/sqlite-performance-tuning/
// and https://kerkour.com/sqlite-for-servers.
var defaultPragmas = map[string]string{
	"journal_mode": "WAL",
	"synchronous":  "NORMAL",
	"temp_store":   "MEMORY",
	"cache_size":   "1000000000",
	"foreign_keys": "true",
	"mmap_size":    "2147483648", // 2 GiB
}

func buildDSN(path string, readOnly bool, busyTimeout time.Duration, pragmaOverrides map[string]string) string {
	query := url.Values{}
	if readOnly {
		query.Add("_txlock", "deferred")
	} else {
		query.Add("_txlock", "immediate")
	}
	query.Add("_time_format", "sqlite")
	query.Add("_busy_timeout", fmt.Sprint(busyTimeout.Milliseconds()))
	if readOnly {
		query.Add("mode", "ro")
	} else {
		query.Add("mode", "rwc")
	}

	pragmas := make(map[string]string, len(defaultPragmas))
	maps.Copy(pragmas, defaultPragmas)
	maps.Copy(pragmas, pragmaOverrides)
	for k, v := range pragmas {
		query.Add("_pragma", k+"="+v)
	}

	return fmt.Sprintf("file:%s?%s", path, query.Encode())
}

// DB contains both a rw and a ro database conn.
type DB struct {
	// Read/write conn.
	rw *sql.DB
	// Read only conn.
	ro *sql.DB
}

// QueryRO returns a [Queries] object, with a read only connection.
func (d *DB) QueryRO() *Queries {
	return New(d.ro)
}

// QueryRW returns a [Queries] object, with a read/write connection.
func (d *DB) QueryRW() *Queries {
	return New(d.rw)
}

// RW returns the underlying read/write [*sql.DB] for raw SQL operations
// that cannot be expressed through sqlc-generated queries.
func (d *DB) RW() *sql.DB {
	return d.rw
}

// RO returns the underlying read-only [*sql.DB] connection pool.
func (d *DB) RO() *sql.DB {
	return d.ro
}

// WithRWTx runs the given function in a read/write transaction.
// Unlike [WithTx], this passes a raw [*sql.Tx] so callers can wrap it
// with any sqlc-generated Queries type (e.g. from sub-packages).
func (d *DB) WithRWTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := d.rw.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			logg.Warn(ctx, "failed to rollback rw transaction", "err", rbErr)
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// BeginTX returns a Queries object, with a read/write transaction connection.
// You must call [Commit] or [Rollback] on the returned object when done.
func (d *DB) BeginTX(ctx context.Context) (*Queries, error) {
	tx, err := d.rw.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	return New(d.rw).WithTx(tx), nil
}

// WithTx runs the given function in a transaction.
// If the function returns an error, the transaction is rolled back.
func (d *DB) WithTx(ctx context.Context, fn func(q *Queries) error) error {
	tx, err := d.BeginTX(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			logg.Warn(ctx, "failed to rollback transaction", "err", rbErr)
		}
	}()

	err = fn(tx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Commit the tx.
func (q *Queries) Commit() error {
	tx, ok := q.db.(*sql.Tx)
	if !ok {
		return errors.New("not a tx")
	}
	return tx.Commit()
}

// Rollback the tx.
func (q *Queries) Rollback() error {
	tx, ok := q.db.(*sql.Tx)
	if !ok {
		return errors.New("not a tx")
	}
	return tx.Rollback()
}

// Close closes both conns.
func (d *DB) Close() error {
	err0 := d.ro.Close()
	err1 := d.rw.Close()
	if err0 != nil || err1 != nil {
		return fmt.Errorf("closing failed: %s, %s", err0, err1)
	}
	return nil
}

// Open opens the database with the given migration files.
// To deal with concurrency efficiently, we maintain both a read-only connection pool,
// and a read/write "pool" with only one connection in it.
// You should maintain only a single [*DB] object per SQLite file at a time in your application.
// You must call [*DB.Close] when the conn is no longer needed.
// Optional pragmaOverrides replace matching keys in the default pragma set.
//
// Parameters:
//   - ctx: context for logging and migration execution.
//   - path: filesystem path for the SQLite database file.
//   - migrationsFS: embedded filesystem containing migration SQL files.
//   - migrationsDir: subdirectory within migrationsFS that holds the migration files.
//   - pragmaOverrides: optional map of SQLite pragma overrides (at most one).
func Open(ctx context.Context, path string, migrationsFS embed.FS, migrationsDir string, pragmaOverrides ...map[string]string) (db *DB, err error) {
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0750)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
	}

	var overrides map[string]string
	if len(pragmaOverrides) > 0 {
		overrides = pragmaOverrides[0]
	}

	// Open read/write conn.
	rw, err := sql.Open(driver, buildDSN(path, false, time.Second*5, overrides))
	if err != nil {
		return nil, fmt.Errorf("failed to open db (rw): %w", err)
	}
	rw.SetMaxOpenConns(1)

	// Open read only conn.
	ro, err := sql.Open(driver, buildDSN(path, true, time.Second*5, overrides))
	if err != nil {
		rw.Close()
		return nil, fmt.Errorf("failed to open db (ro): %w", err)
	}
	ro.SetMaxOpenConns(max(4, runtime.NumCPU()))

	// Close both connections if any subsequent step fails.
	defer func() {
		if err != nil {
			rw.Close()
			ro.Close()
		}
	}()

	cwd, _ := os.Getwd()
	logg.Info(ctx, "opened database", "path", path, "cwd", cwd)

	// Prepare and run migrations.
	files, err := fs.Sub(migrationsFS, migrationsDir)
	if err != nil {
		return nil, errors.New("no migrations")
	}

	dbLogger := slog.NewLogLogger(logg.GetLogger(ctx).Handler(), slog.LevelInfo)
	provider, err := goose.NewProvider(dialect, rw, files,
		goose.WithLogger(dbLogger),
		goose.WithVerbose(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrations: %w", err)
	}

	logg.Info(ctx, "running migrations")
	_, err = provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{rw: rw, ro: ro}, nil
}
