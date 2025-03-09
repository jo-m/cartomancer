package main

import (
	"context"
	"fmt"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logg"
	"goweb/internal/pkg/password"
	"goweb/internal/pkg/session"
	"goweb/internal/pkg/svc"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type config struct {
	logg.Config

	HTTPListenAddr string `arg:"--listen-addr,env:LISTEN_ADDR" help:"TCP address to listen at for HTTP requests" placeholder:"HOST:PORT" default:"127.0.0.1:8050"`
	DBPath         string `arg:"--db-path,env:DB_PATH" help:"Path where the SQLite database will be stored" placeholder:"PATH" default:"data/db.sqlite"`
}

// TODO: Those are for argparse
func (c config) Version() string     { return "Version" }
func (c config) Description() string { return "Description" }
func (c config) Epilogue() string    { return "Epilogue" }

func NewHandler(q *db.DB, logger *slog.Logger) http.Handler {
	logger = logger.With("mod", "svc")

	sess := session.NewStore(q, session.Config{
		MaxIdleTimeout:     time.Minute * 20,
		MaxAbsoluteTimeout: time.Hour,
		CookieName:         "sid",
		CookiePath:         "/",
	})
	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(logg.AttachLogger(logger))
	mux.Use(logg.RequestLogger)
	mux.Use(middleware.RequestSize(1024 * 1024))
	mux.Use(middleware.StripSlashes)
	mux.Use(sess.Middleware())
	mux.Use(middleware.Recoverer)

	mux.Mount("/", svc.New(q, sess))

	return mux
}

func createUser(ctx context.Context, q *db.Queries, email, pass string) error {
	uid, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to create uuid: %w", err)
	}
	_, err = q.CreateUser(ctx, db.CreateUserParams{
		ID:           uid.String(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Email:        email,
		Name:         "test",
		PasswordHash: password.Hashed(pass),
	})
	return err
}

func main() {
	// Parse args.
	c := config{}
	p, err := arg.NewParser(arg.Config{Out: os.Stderr}, &c)
	if err != nil {
		panic(fmt.Sprintf("failed to create parser: %s", err))
	}
	p.MustParse(os.Args[1:])

	// Initialize logging.
	logger := slog.New(logg.NewHandler(c.Config))
	slog.SetDefault(logger)
	ctx := logg.WithLogger(context.Background(), logger)

	// Migrations.
	q, err := db.Open(ctx, c.DBPath)
	if err != nil {
		logg.Panic(ctx, "Failed to open db", "err", err)
	}
	defer q.Close()

	tx, err := q.QueryTX(ctx)
	defer tx.Rollback()
	if err != nil {
		logg.Panic(ctx, "Failed to begin tx", "err", err)
	}
	createUser(ctx, tx, "test@example.org", "asdf")
	if err != nil {
		logg.Error(ctx, "CreateUser failed", "err", err)
	}
	for range 10 {
		createUser(ctx, tx, gofakeit.Email(), password.GenRandPrintableString(32))
	}
	err = tx.Commit()
	if err != nil {
		logg.Panic(ctx, "Failed to commit", "err", err)
	}

	s := &http.Server{
		Addr:              c.HTTPListenAddr,
		Handler:           NewHandler(q, logger),
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	logg.Warn(ctx, gofakeit.HackerPhrase())
	logg.Info(ctx, "Listening", "addr", s.Addr)
	logg.Error(ctx, "ListenAndServe failed", "err", s.ListenAndServe())
}
