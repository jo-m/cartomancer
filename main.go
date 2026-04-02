// Cartomancer runs the web server and job runner.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"jo-m.ch/go/cartomancer/internal/pkg/api"
	"jo-m.ch/go/cartomancer/internal/pkg/app"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/forecast"
	"jo-m.ch/go/cartomancer/internal/pkg/geonames"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/mail"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo"
	"jo-m.ch/go/cartomancer/internal/pkg/password"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures"
	"jo-m.ch/go/cartomancer/internal/pkg/segment"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
	"jo-m.ch/go/cartomancer/internal/pkg/trackgroup"
	"jo-m.ch/go/cartomancer/internal/pkg/users"
)

// serveCmd starts the web server and job runner.
type serveCmd struct{}

// setpassCmd sets the password for any user by email.
type setpassCmd struct {
	Email    string `arg:"positional,required" help:"email address of the user"`
	Password string `arg:"positional" help:"new password (generated if omitted)"`
}

// deletealltracksCmd deletes all tracks and associated data from the database.
type deletealltracksCmd struct{}

type config struct {
	Serve           *serveCmd           `arg:"subcommand:serve" help:"start the web server and background jobs (default)"`
	Setpass         *setpassCmd         `arg:"subcommand:setpass" help:"set password for a user"`
	Deletealltracks *deletealltracksCmd `arg:"subcommand:deletealltracks" help:"delete all tracks (dev mode only)"`

	HTTPListenAddr string `arg:"--listen-addr,env:LISTEN_ADDR" help:"TCP address to listen at for HTTP requests" placeholder:"HOST:PORT" default:"127.0.0.1:8080"`
	DBPath         string `arg:"--db-path,env:DB_PATH" help:"Path where the SQLite database will be stored" placeholder:"PATH" default:"data/db.sqlite"`

	logg.LoggConfig
	app.AppConfig
	session.SessionConfig
	mail.MailerConfig
	jobs.JobsConfig
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

	apiHandler, err := api.New(d, sess, jobSubmitter, appConfig)
	if err != nil {
		logg.Panic(ctx, "Failed to create API handler", "err", err)
	}
	mux.Mount("/api", apiHandler)
	mux.Get("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
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
		plainPass = password.GenRandAlnumString(password.GeneratedPasswordLen)
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
		plainPass = password.GenRandAlnumString(password.GeneratedPasswordLen)
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

// runDeleteAllTracks deletes all tracks, groups, segments, and associated data from the database.
// Refuses to run unless development mode is enabled.
func runDeleteAllTracks(ctx context.Context, d *db.DB, appCfg app.AppConfig) {
	if !appCfg.DevelopmentMode {
		logg.Panic(ctx, "deletealltracks requires development mode (--app-dev-mode / APP_DEV_MODE)")
	}

	err := d.WithTx(ctx, func(q *db.Queries) error {
		if txErr := q.DeleteAllSegmentTracks(ctx); txErr != nil {
			return fmt.Errorf("delete segment tracks: %w", txErr)
		}
		if txErr := q.DeleteAllSegments(ctx); txErr != nil {
			return fmt.Errorf("delete segments: %w", txErr)
		}
		if txErr := q.DeleteAllSegmentJunctions(ctx); txErr != nil {
			return fmt.Errorf("delete segment junctions: %w", txErr)
		}
		if txErr := q.DeleteAllTrackGroups(ctx); txErr != nil {
			return fmt.Errorf("delete track groups: %w", txErr)
		}
		return nil
	})
	if err != nil {
		logg.Panic(ctx, "Failed to delete segments and groups", "err", err)
	}
	logg.Warn(ctx, "Deleted all segments and groups")

	n, err := d.QueryRW().DeleteAllTracks(ctx)
	if err != nil {
		logg.Panic(ctx, "Failed to delete tracks", "err", err)
	}
	logg.Warn(ctx, "Deleted all tracks", "count", n)
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

	// In demo mode, use a separate database file to avoid overwriting production data.
	if c.DemoMode {
		c.DBPath += ".demo"
		logg.Warn(ctx, "demo mode: using separate database", "path", c.DBPath)
	}

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

	// Handle deletealltracks subcommand early, before server-specific setup.
	if c.Deletealltracks != nil {
		runDeleteAllTracks(ctx, d, c.AppConfig)
		return
	}

	// Everything below is for the serve subcommand (default).

	if !c.MailerConfig.Enabled() {
		logg.Error(ctx, "Email sending is disabled (MAIL_SMTP_HOST, MAIL_SMTP_PORT, or MAIL_FROM not set)")
	}
	if c.SessionConfig.JWTSecret == "" {
		logg.Error(ctx, "SESSION_JWT_SECRET not set, using a random ephemeral secret (sessions will not survive restarts)")
	}
	if c.AppConfig.EmailJWTSecret == "" {
		logg.Error(ctx, "APP_EMAIL_JWT_SECRET not set, using a random ephemeral secret (pending email verifications will not survive restarts)")
	}

	// Insert test user in development or demo mode (must happen before demo triggers).
	if c.DevelopmentMode || c.DemoMode {
		created, _, err := ensureInitialAdmin(ctx, d, app.DevInitialAdminEmail, app.DevInitialAdminPassword)
		if err != nil {
			logg.Warn(ctx, "Failed to create dev user", "err", err)
		} else if created {
			logg.Info(ctx, "Created dev user", "email", app.DevInitialAdminEmail, "password", app.DevInitialAdminPassword)
		}
	}

	// Install demo mode triggers to lock down user tables.
	if c.DemoMode {
		if err := app.InstallDemoTriggers(ctx, d.RW()); err != nil {
			logg.Panic(ctx, "Failed to install demo triggers", "err", err)
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
	jobs.Periodic(ctxJobs, w.Submitter(), c.GetCleanerArgs(), time.Minute, false)

	jobs.MustRegisterJob(w, mail.NewMailer(c.MailerConfig))
	jobs.MustRegisterJob(w, users.NewEmailVerificationCleaner(d))
	jobs.Periodic(ctxJobs, w.Submitter(), users.EmailVerificationCleanerArgs(), time.Hour, false)

	jobs.MustRegisterJob(w, meteo.NewDownloader(d))
	jobs.Periodic(ctxJobs, w.Submitter(), meteo.DownloaderArgs{}, time.Hour, true)
	jobs.MustRegisterJob(w, meteo.NewCleaner(d))
	jobs.Periodic(ctxJobs, w.Submitter(), meteo.CleanerArgs(), time.Hour, false)

	jobs.MustRegisterJob(w, forecast.NewSummarizer(d))
	jobs.Periodic(ctxJobs, w.Submitter(), forecast.SummarizerArgs{}, time.Hour, true)

	jobs.MustRegisterJob(w, geonames.NewDownloader(d))
	jobs.Periodic(ctxJobs, w.Submitter(), geonames.DownloaderArgs{}, 7*24*time.Hour, true)
	jobs.MustRegisterJob(w, geonames.NewLabeler(d))

	jobs.MustRegisterJob(w, trackgroup.NewGrouper(d))

	jobs.MustRegisterJob(w, segment.NewBuilder(d))

	jobs.MustRegisterJob(w, roadclosures.NewDownloader(d))
	jobs.Periodic(ctxJobs, w.Submitter(), roadclosures.DownloaderArgs{}, 24*time.Hour, true)

	if !c.DemoMode {
		jobs.MustRegisterJob(w, db.NewBackup(d, c.DBPath))
		jobs.Periodic(ctxJobs, w.Submitter(), db.BackupArgs{}, 24*time.Hour, false)
	}

	if c.DemoMode {
		jobs.MustRegisterJob(w, app.NewDemoTrackPurger(d))
		jobs.Periodic(ctxJobs, w.Submitter(), app.DemoTrackPurgeArgs{}, app.DemoTrackPurgePeriod, false)
		logg.Warn(ctx, "demo mode is active: users are locked, tracks will be purged periodically")
	}

	// Start job runner and HTTP server.

	ctxShutdown, stop := signal.NotifyContext(ctxJobs, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w.RunInBackground(ctxShutdown)

	s := &http.Server{
		Addr:              c.HTTPListenAddr,
		Handler:           newHandler(ctx, d, c.SessionConfig, c.AppConfig, w.Submitter()),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      20 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		<-ctxShutdown.Done()
		logg.Info(ctx, "Shutting down HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			logg.Error(ctx, "HTTP server shutdown error", "err", err)
		}
	}()

	logg.Info(ctx, "Listening on", "url", fmt.Sprintf("http://%s", s.Addr))
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logg.Error(ctx, "ListenAndServe failed", "err", err)
	}
	logg.Info(ctx, "Server stopped")
}
