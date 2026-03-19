// Detour runs the web server and job runner.
package main

import (
	"context"
	"database/sql"
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
	"jo-m.ch/go/detour/internal/pkg/app"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/geocode"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/mail"
	"jo-m.ch/go/detour/internal/pkg/meteo"
	"jo-m.ch/go/detour/internal/pkg/password"
	"jo-m.ch/go/detour/internal/pkg/rest"
	"jo-m.ch/go/detour/internal/pkg/session"
	"jo-m.ch/go/detour/internal/pkg/trackgroup"
	"jo-m.ch/go/detour/internal/pkg/users"
)

// serveCmd starts the web server and job runner.
type serveCmd struct{}

// setpassCmd sets the password for any user by email.
type setpassCmd struct {
	Email    string `arg:"positional,required" help:"email address of the user"`
	Password string `arg:"positional" help:"new password (generated if omitted)"`
}

type config struct {
	Serve   *serveCmd   `arg:"subcommand:serve" help:"start the web server and background jobs (default)"`
	Setpass *setpassCmd `arg:"subcommand:setpass" help:"set password for a user"`

	logg.LoggConfig
	jobs.JobsConfig
	session.SessionConfig
	mail.MailerConfig
	app.AppConfig

	HTTPListenAddr string `arg:"--listen-addr,env:LISTEN_ADDR" help:"TCP address to listen at for HTTP requests" placeholder:"HOST:PORT" default:"127.0.0.1:8080"`
	DBPath         string `arg:"--db-path,env:DB_PATH" help:"Path where the SQLite database will be stored" placeholder:"PATH" default:"data/db.sqlite"`
}

// validate checks all embedded configs for basic errors.
func (c *config) validate() error {
	if c.HTTPListenAddr == "" {
		return fmt.Errorf("listen address must not be empty")
	}
	if c.DBPath == "" {
		return fmt.Errorf("database path must not be empty")
	}
	for _, err := range []error{
		c.LoggConfig.Validate(),
		c.AppConfig.Validate(),
		c.SessionConfig.Validate(),
		c.JobsConfig.Validate(),
		c.MailerConfig.Validate(),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

// validateProduction checks that all settings required for a production deployment are set.
// Returns nil when DevelopmentMode is enabled.
func (c *config) validateProduction() error {
	if c.DevelopmentMode {
		return nil
	}
	for _, err := range []error{
		c.AppConfig.ValidateProduction(),
		c.SessionConfig.ValidateProduction(),
		c.MailerConfig.ValidateProduction(),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

func newHandler(ctx context.Context, d *db.DB, sessConfig session.SessionConfig, appConfig app.AppConfig, jobSubmitter *jobs.Submitter) http.Handler {
	logger := logg.GetLogger(ctx).With("mod", "svc")

	sess, err := session.NewStore(d, sessConfig, appConfig)
	if err != nil {
		logg.Panic(ctx, "Failed to create session store", "err", err)
	}

	staticFS, err := getStaticFS(appConfig.DevelopmentMode)
	if err != nil {
		logg.Panic(ctx, "Failed to get static files", "err", err)
	}

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(logg.AttachLogger(logger))
	mux.Use(logg.RequestLogger)
	mux.Use(middleware.RequestSize(5 * 1024 * 1024))
	mux.Use(middleware.Compress(5))
	mux.Use(sess.Middleware)
	mux.Use(middleware.Recoverer)

	apiHandler, err := rest.New(d, sess, jobSubmitter, appConfig)
	if err != nil {
		logg.Panic(ctx, "Failed to create API handler", "err", err)
	}
	mux.Mount("/api", apiHandler)
	mux.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /\nAllow: /$\n"))
	})
	mux.Handle("/*", spaHandler(staticFS))

	return mux
}

// ensureInitialAdmin creates an admin account with the given email if none exists yet.
// If plainPass is empty, a random password is generated.
// Returns whether the account was created and the plaintext password.
func ensureInitialAdmin(ctx context.Context, d *db.DB, email, plainPass string) (bool, string, error) {
	_, err := d.QueryRO().GetUserByEmail(ctx, email)
	if err == nil {
		return false, "", nil
	}
	if err != sql.ErrNoRows {
		return false, "", fmt.Errorf("failed to look up user: %w", err)
	}

	if plainPass == "" {
		plainPass = password.GenRandPrintableString(24)
	}

	err = d.WithTx(ctx, func(tx *db.Queries) error {
		uid, txErr := uuid.NewV7()
		if txErr != nil {
			return fmt.Errorf("failed to create uuid: %w", txErr)
		}
		hash, txErr := password.Hash(plainPass)
		if txErr != nil {
			return fmt.Errorf("failed to hash password: %w", txErr)
		}
		_, txErr = tx.CreateUser(ctx, db.CreateUserParams{
			Uuid:           uid.String(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			Email:          email,
			Name:           "Admin",
			PasswordHash:   hash,
			Admin:          1,
			EmailConfirmed: 1,
		})
		return txErr
	})
	if err != nil {
		return false, "", fmt.Errorf("failed to create initial admin: %w", err)
	}
	return true, plainPass, nil
}

// runSetpass sets or resets a user's password by email.
func runSetpass(ctx context.Context, d *db.DB, cmd *setpassCmd) {
	user, err := d.QueryRO().GetUserByEmail(ctx, cmd.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			logg.Panic(ctx, "User not found", "email", cmd.Email)
		}
		logg.Panic(ctx, "Failed to look up user", "err", err)
	}

	plainPass := cmd.Password
	if plainPass == "" {
		plainPass = password.GenRandPrintableString(24)
	}

	hash, err := password.Hash(plainPass)
	if err != nil {
		logg.Panic(ctx, "Failed to hash password", "err", err)
	}

	err = d.WithTx(ctx, func(q *db.Queries) error {
		_, txErr := q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
			UpdatedAt:    time.Now().UTC(),
			PasswordHash: hash,
			Uuid:         user.Uuid,
		})
		if txErr != nil {
			return txErr
		}
		_, txErr = q.DeleteAllUserSessions(ctx, sql.NullString{Valid: true, String: user.Uuid})
		return txErr
	})
	if err != nil {
		logg.Panic(ctx, "Failed to set password", "err", err)
	}

	logg.Warn(ctx, "Password set successfully", "email", cmd.Email, "password", plainPass)
}

func main() {
	ctx := context.Background()

	// Parse args.
	c := config{}
	//revive:disable:superfluous-else
	if p, err := arg.NewParser(arg.Config{Out: os.Stderr}, &c); err != nil {
		panic(err)
	} else {
		p.MustParse(os.Args[1:])
	}

	if err := c.validate(); err != nil {
		panic(fmt.Sprintf("invalid configuration: %s", err))
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

	// Handle setpass subcommand early, before server-specific setup.
	if c.Setpass != nil {
		runSetpass(ctx, d, c.Setpass)
		return
	}

	// Everything below is for the serve subcommand (default).
	if err := c.validateProduction(); err != nil {
		panic(fmt.Sprintf("invalid production configuration: %s", err))
	}

	// Insert test user in development mode.
	if c.DevelopmentMode {
		created, _, err := ensureInitialAdmin(ctx, d, "test@example.org", "asdf")
		if err != nil {
			logg.Warn(ctx, "Failed to create test user", "err", err)
		} else if created {
			logg.Info(ctx, "Created dev admin account", "email", "test@example.org", "password", "asdf")
		}
	}

	// Create initial admin account for production if configured.
	if c.InitAdminEmail != "" && !c.DevelopmentMode {
		created, pass, err := ensureInitialAdmin(ctx, d, c.InitAdminEmail, "")
		if err != nil {
			logg.Panic(ctx, "Failed to create initial admin", "err", err)
		} else if created {
			logg.Warn(ctx, "Created initial admin account -- save the password now, it will not be shown again",
				"email", c.InitAdminEmail, "password", pass)
		}
	}

	// Jobs.
	ctxJobs := logg.WithLogger(ctx, logger.With("mod", "jobs"))
	w, err := jobs.NewWorkers(ctxJobs, d, c.JobsConfig)
	if err != nil {
		logg.Panic(ctx, "Failed to initialize workers", "err", err)
	}

	jobs.MustRegisterJob(w, session.NewCleaner(d))
	jobs.MustRegisterJob(w, mail.NewMailer(c.MailerConfig))
	jobs.MustRegisterJob(w, users.NewEmailVerificationCleaner(d))
	jobs.MustRegisterJob(w, meteo.NewDownloader(d))
	jobs.MustRegisterJob(w, meteo.NewCleaner(d))
	jobs.MustRegisterJob(w, geocode.NewDownloader(d))
	jobs.MustRegisterJob(w, geocode.NewLabeler(d))
	jobs.MustRegisterJob(w, trackgroup.NewGrouper(d))
	jobs.Periodic(ctxJobs, w.Submitter(), c.GetCleanerArgs(), time.Minute, false)
	jobs.Periodic(ctxJobs, w.Submitter(), users.EmailVerificationCleanerArgs(), time.Hour, false)
	jobs.Periodic(ctxJobs, w.Submitter(), meteo.DownloaderArgs{}, time.Hour, true)
	jobs.Periodic(ctxJobs, w.Submitter(), meteo.CleanerArgs(), time.Hour, false)
	jobs.Periodic(ctxJobs, w.Submitter(), geocode.DownloaderArgs{}, 7*24*time.Hour, true)

	// TODO: clean shutdown via context.
	w.RunInBackground(ctxJobs)

	{
		s := &http.Server{
			Addr:              c.HTTPListenAddr,
			Handler:           newHandler(ctx, d, c.SessionConfig, c.AppConfig, w.Submitter()),
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       20 * time.Second,
			WriteTimeout:      20 * time.Second,
			MaxHeaderBytes:    1 << 20,
		}
		logg.Warn(ctx, gofakeit.HackerPhrase())
		logg.Info(ctx, "Listening on", "url", fmt.Sprintf("http://%s", s.Addr))
		logg.Error(ctx, "ListenAndServe failed", "err", s.ListenAndServe())
	}
}
