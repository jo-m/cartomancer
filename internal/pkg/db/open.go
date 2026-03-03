package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
	"jo-m.ch/go/detour/internal/pkg/logg"
	_ "modernc.org/sqlite" // Database driver.
)

const (
	driver     = "sqlite"
	dialect    = "sqlite3"
	migrations = "migrations"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func buildDSN(path string, readOnly bool, busyTimeout time.Duration) string {
	query := url.Values{}
	query.Add("_txlock", "deferred")
	query.Add("_time_format", "sqlite")
	query.Add("_busy_timeout", fmt.Sprint(busyTimeout.Milliseconds()))
	if readOnly {
		query.Add("mode", "ro")
	} else {
		query.Add("mode", "rwc")
	}

	// See https://phiresky.github.io/blog/2020/sqlite-performance-tuning/.
	pragmas := map[string]string{
		"journal_mode": "WAL",
		"synchronous":  "NORMAL",
		"temp_store":   "MEMORY",
		"foreign_keys": "true",
		"mmap_size":    "2147483648", // 2 GiB
	}
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
	defer tx.Rollback()

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

// Open opens the database.
// To deal with concurrency efficiently, we maintain both a read-only connection pool,
// and a read/write "pool" with only one connection in it.
// You should maintain only a single [*DB] object per SQlite file at a time in your application.
// You must call [*DB.Close] when the conn is no longer needed.
func Open(ctx context.Context, path string) (db *DB, err error) {
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0750)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
	}

	// Open read/write conn.
	rw, err := sql.Open(driver, buildDSN(path, false, time.Second*5))
	if err != nil {
		return nil, fmt.Errorf("failed to open db (rw): %w", err)
	}
	rw.SetMaxOpenConns(1)

	// Open read only conn.
	ro, err := sql.Open(driver, buildDSN(path, true, time.Second*5))
	if err != nil {
		return nil, fmt.Errorf("failed to open db (ro): %w", err)
	}

	// Prepare and run migrations.
	files, err := fs.Sub(embedMigrations, migrations)
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

	logg.Info(ctx, "Running migrations.")
	_, err = provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{rw: rw, ro: ro}, nil
}
