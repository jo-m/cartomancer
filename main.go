package main

import (
	"context"
	"database/sql"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/password"
	"goweb/internal/pkg/svc"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "modernc.org/sqlite"
)

func NewHandler(db *sql.DB) http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Timeout(60 * time.Second))

	mux.Mount("/", svc.New(db))

	return mux
}

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
		Email:        "test@example.org",
		Name:         "test",
		PasswordHash: password.Hashed("asdf"),
		Biography:    sql.NullString{},
	})
	if err != nil {
		slog.Error("CreateUser failed", "err", err)
	}

	s := &http.Server{
		Addr:              os.Getenv("HTTP_LISTEN"),
		Handler:           NewHandler(d),
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	slog.Info("Listening", "addr", s.Addr)
	slog.Error("ListenAndServe failed", "err", s.ListenAndServe())
}
