package main

import (
	"context"
	"database/sql"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logging"
	"goweb/internal/pkg/password"
	"goweb/internal/pkg/session"
	"goweb/internal/pkg/svc"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "modernc.org/sqlite"
)

func NewHandler(db *sql.DB, logger *slog.Logger) http.Handler {
	logger = logger.With("mod", "svc")

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(logging.AttachLogger(logger))
	mux.Use(logging.RequestLogger)
	mux.Use(middleware.RequestSize(1024 * 1024))
	mux.Use(middleware.StripSlashes)
	mux.Use(session.Middleware(session.Config{
		DB:         db,
		MaxAge:     time.Second * 1800,
		CookieName: "s",
		CookiePath: "/",
	}))
	mux.Use(middleware.Recoverer)

	mux.Mount("/", svc.New(db))

	return mux
}

func main() {
	logger := slog.New(logging.NewHandler())
	slog.SetDefault(logger)
	ctx := logging.WithLogger(context.Background(), logger)

	// Migrations.
	// TODO: real config via flags/env vars
	d, err := db.Open(ctx, os.Getenv("GOOSE_DBSTRING"))
	if err != nil {
		logging.Panic(ctx, "Failed to open db", "err", err)
	}
	defer d.Close()

	q := db.New(d)
	_, err = q.CreateUser(ctx, db.CreateUserParams{
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Email:        "test@example.org",
		Name:         "test",
		PasswordHash: password.Hashed("asdf"),
	})
	if err != nil {
		logger.Error("CreateUser failed", "err", err)
	}

	s := &http.Server{
		Addr:              os.Getenv("HTTP_LISTEN"),
		Handler:           NewHandler(d, logger),
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	logger.Info("Listening", "addr", s.Addr)
	logger.Error("ListenAndServe failed", "err", s.ListenAndServe())
}
