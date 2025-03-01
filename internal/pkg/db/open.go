package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"net/url"

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

func buildDSN(path string, readOnly bool) string {
	query := url.Values{}
	query.Add("_txlock", "deferred")
	query.Add("_time_format", "sqlite")
	if readOnly {
		query.Add("mode", "ro")
	} else {
		query.Add("mode", "rwc")
	}

	// TODO: https://phiresky.github.io/blog/2020/sqlite-performance-tuning/
	pragmas := map[string]string{
		"journal_mode": "WAL",
		"locking_mode": "NORMAL",
		"foreign_keys": "true",
	}
	for k, v := range pragmas {
		query.Add("_pragma", k+"="+v)
	}

	return fmt.Sprintf("file:%s?%s", path, query.Encode())
}

func Open(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open(driver, buildDSN(path, false))
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	goose.SetDialect(dialect)
	goose.SetBaseFS(embedMigrations)
	slog.Info("Running migrations.")
	if err := goose.UpContext(ctx, db, migrations); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	// TODO: replace with proper log statement
	if err := goose.VersionContext(ctx, db, migrations); err != nil {
		return nil, fmt.Errorf("failed to get migration version: %w", err)
	}

	return db, nil
}
