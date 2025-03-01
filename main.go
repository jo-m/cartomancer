package main

import (
	"context"
	"database/sql"
	"goweb/internal/pkg/api"
	"goweb/internal/pkg/db"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	ctx := context.Background()

	// Migrations.
	// TODO: real config via flags/env vars
	d, err := db.Open(ctx, os.Getenv("GOOSE_DBSTRING"))
	if err != nil {
		panic(err)
	}
	defer d.Close()

	q := db.New(d)
	_, err = q.CreateUser(ctx, db.CreateUserParams{
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Username:     "test",
		Email:        "test@example.org",
		PasswordHash: "test",
		Biography:    sql.NullString{},
	})
	if err != nil {
		slog.Error("CreateUser failed", "err", err)
	}

	s := &http.Server{
		Addr:              os.Getenv("HTTP_LISTEN"),
		Handler:           api.New(d),
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	slog.Info("Listening", "addr", s.Addr)
	slog.Error("ListenAndServe failed", "err", s.ListenAndServe())
}
