// Goweb runs the web server and job runner.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jo-m/goweb/internal/pkg/app"
	"github.com/jo-m/goweb/internal/pkg/db"
	"github.com/jo-m/goweb/internal/pkg/endpoints"
	"github.com/jo-m/goweb/internal/pkg/jobs"
	"github.com/jo-m/goweb/internal/pkg/logg"
	"github.com/jo-m/goweb/internal/pkg/mail"
	"github.com/jo-m/goweb/internal/pkg/oapi"
	"github.com/jo-m/goweb/internal/pkg/password"
	"github.com/jo-m/goweb/internal/pkg/session"
	"github.com/jo-m/goweb/internal/pkg/utl"
)

type config struct {
	logg.LoggConfig
	jobs.JobsConfig
	session.SessionConfig
	mail.MailerConfig
	app.AppConfig

	HTTPListenAddr string `arg:"--listen-addr,env:LISTEN_ADDR" help:"TCP address to listen at for HTTP requests" placeholder:"HOST:PORT" default:"127.0.0.1:8050"`
	DBPath         string `arg:"--db-path,env:DB_PATH" help:"Path where the SQLite database will be stored" placeholder:"PATH" default:"data/db.sqlite"`
}

func newHandler(ctx context.Context, d *db.DB, sessConfig session.SessionConfig, appConfig app.AppConfig) http.Handler {
	logger := logg.GetLogger(ctx).With("mod", "svc")
	mux := chi.NewRouter()

	oapi.PrintRoutes(ctx)

	{

		sess, err := session.NewStore(d, sessConfig, appConfig)
		if err != nil {
			logg.Panic(ctx, "Failed to create session store", "err", err)
		}

		svcMux := chi.NewRouter()
		svcMux.Use(middleware.RequestID)
		svcMux.Use(logg.AttachLogger(logger))
		svcMux.Use(logg.RequestLogger)
		svcMux.Use(middleware.RequestSize(1024 * 1024))
		svcMux.Use(middleware.Compress(5))
		svcMux.Use(middleware.RedirectSlashes)
		svcMux.Use(sess.Middleware)
		svcMux.Use(middleware.Recoverer)
		svcMux.Use(oapi.AttachLinks(oapi.Links{
			Base: appConfig.ExternalBaseURL,
		}))

		svcMux.Mount("/", endpoints.New(d, sess))
		mux.Mount("/", svcMux)
	}

	{
		staticFS, err := getStaticFS()
		if err != nil {
			logg.Panic(ctx, "Failed to get static files", "err", err)
		}

		staticMux := chi.NewRouter()
		staticMux.Use(middleware.RequestID)
		staticMux.Use(logg.AttachLogger(logger))
		staticMux.Use(logg.RequestLogger)

		staticMux.Mount("/", http.StripPrefix("/static", http.FileServerFS(staticFS)))
		mux.Mount("/static", staticMux)
	}

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
	ctx := context.Background()

	// Parse args.
	c := config{}
	if p, err := arg.NewParser(arg.Config{Out: os.Stderr}, &c); err != nil {
		panic(err)
	} else {
		p.MustParse(os.Args[1:])
	}

	// Initialize logging.
	logger := logg.New(c.LoggConfig)
	if c.DevelopmentMode {
		logg.DisableDefaultLogger()
	} else {
		slog.SetDefault(logger)
	}
	ctx = logg.WithLogger(ctx, logg.New(c.LoggConfig))

	// Migrations.
	d, err := db.Open(ctx, c.DBPath)
	if err != nil {
		logg.Panic(ctx, "Failed to open db", "err", err)
	}
	defer d.Close()

	// Insert users.
	{
		err := d.WithTx(ctx, func(tx *db.Queries) error {
			_ = createUser(ctx, tx, "test@example.org", "asdf")
			if err != nil {
				logg.Error(ctx, "CreateUser failed", "err", err)
			}
			for range 3 {
				_ = createUser(ctx, tx, gofakeit.Email(), password.GenRandPrintableString(32))
			}
			return nil
		})
		if err != nil {
			logg.Panic(ctx, "Failed to commit", "err", err)
		}
	}

	// Jobs.
	{
		ctx := logg.WithLogger(ctx, logger.With("mod", "jobs"))
		w, err := jobs.NewWorkers(ctx, d, c.JobsConfig)
		if err != nil {
			logg.Panic(ctx, "Failed to initialize workers", "err", err)
		}

		jobs.MustRegisterJob(w, session.NewCleaner(d))
		jobs.MustRegisterJob(w, mail.NewMailer(c.MailerConfig))
		jobs.Periodic(ctx, w.Submitter(), c.GetCleanerArgs(), time.Minute)

		for range 10 {
			_ = jobs.Submit(ctx, w.Submitter(), mail.Args{
				To:      []string{gofakeit.Email(), gofakeit.Email(), gofakeit.Email()},
				Subject: gofakeit.Phrase(),
				Body:    utl.Must(gofakeit.EmailText(&gofakeit.EmailOptions{})),
			}, jobs.Params{})
		}

		// TODO: clean shutdown via context.
		w.RunInBackground(ctx)
	}

	{
		s := &http.Server{
			Addr:              c.HTTPListenAddr,
			Handler:           newHandler(ctx, d, c.SessionConfig, c.AppConfig),
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
