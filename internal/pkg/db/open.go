package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"goweb/internal/pkg/logging"
	"io/fs"
	"net/url"
	"strings"

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

type gooseLogger struct {
	ctx context.Context
}

func (g *gooseLogger) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	msg = strings.TrimSpace(msg)
	logging.Panic(g.ctx, msg)
}

func (g *gooseLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	logging.Info(g.ctx, strings.TrimSpace(msg))
}

var _ goose.Logger = (*gooseLogger)(nil)

func Open(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open(driver, buildDSN(path, false))
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	files, err := fs.Sub(embedMigrations, migrations)
	if err != nil {
		return nil, errors.New("no migrations")
	}
	provider, err := goose.NewProvider(dialect, db, files,
		goose.WithLogger(&gooseLogger{ctx: ctx}),
		goose.WithVerbose(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrations: %w", err)
	}

	logging.Info(ctx, "Running migrations.")
	_, err = provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}
