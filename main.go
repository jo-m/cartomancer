package main

import (
	"context"
	"fmt"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/endpoints"
	"goweb/internal/pkg/jobs"
	"goweb/internal/pkg/logg"
	"goweb/internal/pkg/password"
	"goweb/internal/pkg/session"
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

func NewHandler(d *db.DB, logger *slog.Logger, sessConfig session.Config) http.Handler {
	logger = logger.With("mod", "svc")

	sess := session.MakeStore(d, sessConfig)
	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(logg.AttachLogger(logger))
	mux.Use(logg.RequestLogger)
	mux.Use(middleware.RequestSize(1024 * 1024))
	mux.Use(middleware.StripSlashes)
	mux.Use(sess.Middleware)
	mux.Use(middleware.Recoverer)

	mux.Mount("/", endpoints.New(d, sess))

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
		PasswordHash: password.Hash(pass),
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
	d, err := db.Open(ctx, c.DBPath)
	if err != nil {
		logg.Panic(ctx, "Failed to open db", "err", err)
	}
	defer d.Close()

	// Insert users.
	{
		err := d.WithTx(ctx, func(tx *db.Queries) error {
			createUser(ctx, tx, "test@example.org", "asdf")
			if err != nil {
				logg.Error(ctx, "CreateUser failed", "err", err)
			}
			for range 3 {
				createUser(ctx, tx, gofakeit.Email(), password.GenRandPrintableString(32))
			}
			return nil
		})
		if err != nil {
			logg.Panic(ctx, "Failed to commit", "err", err)
		}
	}

	sessionConfig := session.Config{
		MaxIdleTimeout:     time.Minute * 20,
		MaxAbsoluteTimeout: time.Hour,
		CookieName:         "sid",
		CookiePath:         "/",
	}

	// Jobs.
	{
		ctx := logg.WithLogger(ctx, logger.With("mod", "jobs"))
		w, err := jobs.NewWorkers(ctx, d, jobs.Config{MaxParallel: 2, AutoCleanupPeriod: time.Minute})
		if err != nil {
			logg.Panic(ctx, "Failed to initialize workers", "err", err)
		}
		jobs.MustAddWorker(w, &session.Cleaner{})
		jobs.Periodic(ctx, w.Submitter(), 1, sessionConfig.GetCleanerArgs(), time.Minute)
		// TODO: clean shutdown via context.
		w.RunInBackground(ctx)
	}

	{
		s := &http.Server{
			Addr:              c.HTTPListenAddr,
			Handler:           NewHandler(d, logger, sessionConfig),
			ReadHeaderTimeout: 20 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			MaxHeaderBytes:    1 << 20,
		}
		logg.Warn(ctx, gofakeit.HackerPhrase())
		logg.Info(ctx, "Listening on", "url", fmt.Sprintf("http://%s", s.Addr))
		logg.Error(ctx, "ListenAndServe failed", "err", s.ListenAndServe())
	}
}
