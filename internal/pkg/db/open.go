package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"goweb/internal/pkg/logg"
	"io/fs"
	"net/url"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
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

type gooseLogger struct {
	ctx context.Context
}

func (g *gooseLogger) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	msg = strings.TrimSpace(msg)
	logg.Panic(g.ctx, msg)
}

func (g *gooseLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	logg.Info(g.ctx, strings.TrimSpace(msg))
}

var _ goose.Logger = (*gooseLogger)(nil)

type DB struct {
	// Read/write conn.
	rw *sql.DB
	// Read only conn.
	ro *sql.DB
}

func (d *DB) QueryRO() *Queries {
	return New(d.ro)
}

func (d *DB) QueryTX(ctx context.Context) (*Queries, error) {
	tx, err := d.rw.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	return New(d.rw).WithTx(tx), nil
}

func (q *Queries) Commit() error {
	tx, ok := q.db.(*sql.Tx)
	if !ok {
		return errors.New("not a tx")
	}
	return tx.Commit()
}

func (q *Queries) Rollback() error {
	tx, ok := q.db.(*sql.Tx)
	if !ok {
		return errors.New("not a tx")
	}
	return tx.Rollback()
}

func (d *DB) Close() error {
	err := d.ro.Close()
	if err != nil {
		return err
	}
	return d.rw.Close()
}

func Open(ctx context.Context, path string) (db *DB, err error) {
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
	provider, err := goose.NewProvider(dialect, rw, files,
		goose.WithLogger(&gooseLogger{ctx: ctx}),
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
