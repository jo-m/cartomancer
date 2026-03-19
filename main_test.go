package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/app"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/mail"
	"jo-m.ch/go/detour/internal/pkg/password"
	"jo-m.ch/go/detour/internal/pkg/session"
)

func minimalProductionConfig() config {
	return config{
		LoggConfig: logg.LoggConfig{},
		JobsConfig: jobs.JobsConfig{
			AutoCleanupPeriod: time.Minute,
			AutoCleanupMinAge: 5 * time.Minute,
		},
		SessionConfig: session.SessionConfig{
			IdleTimeout:     24 * time.Hour,
			AbsoluteTimeout: 72 * time.Hour,
			CookieName:      "sid",
			JWTSecret:       "some-secret-value-here",
			CookieDomain:    "example.com",
			CookiePath:      "/",
		},
		MailerConfig: mail.MailerConfig{
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			From:     "noreply@example.com",
		},
		AppConfig: app.AppConfig{
			InstanceName:            "Detour",
			ExternalBaseURL:         "https://example.com",
			EmailJWTSecret:          "email-secret",
			EmailVerificationExpiry: 2 * time.Hour,
			TrackColor:              "currentColor",
		},
		HTTPListenAddr: "127.0.0.1:8080",
		DBPath:         "data/db.sqlite",
	}
}

func TestProductionConfigValidates(t *testing.T) {
	c := minimalProductionConfig()
	require.NoError(t, c.validate())
	require.NoError(t, c.validateProduction())
}

func TestProductionConfigRejectsDevMode(t *testing.T) {
	c := minimalProductionConfig()
	c.DevelopmentMode = true
	// validate() still passes.
	require.NoError(t, c.validate())
	// validateProduction() is skipped in dev mode.
	require.NoError(t, c.validateProduction())
}

func TestProductionConfigRejectsMissingFields(t *testing.T) {
	t.Run("missing external base URL", func(t *testing.T) {
		c := minimalProductionConfig()
		c.ExternalBaseURL = ""
		require.ErrorContains(t, c.validateProduction(), "APP_EXTERNAL_BASE_URL")
	})

	t.Run("missing email JWT secret", func(t *testing.T) {
		c := minimalProductionConfig()
		c.EmailJWTSecret = ""
		require.ErrorContains(t, c.validateProduction(), "APP_EMAIL_JWT_SECRET")
	})

	t.Run("missing JWT secret", func(t *testing.T) {
		c := minimalProductionConfig()
		c.JWTSecret = ""
		require.ErrorContains(t, c.validateProduction(), "SESSION_JWT_SECRET")
	})

	t.Run("missing mail from", func(t *testing.T) {
		c := minimalProductionConfig()
		c.From = ""
		require.ErrorContains(t, c.validateProduction(), "MAIL_FROM")
	})

	t.Run("TLS disabled", func(t *testing.T) {
		c := minimalProductionConfig()
		c.SMTPNoTLS = true
		require.ErrorContains(t, c.validateProduction(), "MAIL_SMTP_NO_TLS")
	})
}

func TestEnsureInitialAdmin_CreatesUser(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := t.Context()

	created, pass, err := ensureInitialAdmin(ctx, d, "admin@example.com", "")
	require.NoError(t, err)
	require.True(t, created)
	require.NotEmpty(t, pass)

	// Verify the user exists and the password works.
	user, err := d.QueryRO().GetUserByEmail(ctx, "admin@example.com")
	require.NoError(t, err)
	require.Equal(t, "Admin", user.Name)
	require.Equal(t, int64(1), user.Admin)
	require.Equal(t, int64(1), user.EmailConfirmed)
	require.True(t, password.Check(pass, user.PasswordHash))
}

func TestEnsureInitialAdmin_ExplicitPassword(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := t.Context()

	created, pass, err := ensureInitialAdmin(ctx, d, "admin@example.com", "my-known-pass")
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "my-known-pass", pass)

	user, err := d.QueryRO().GetUserByEmail(ctx, "admin@example.com")
	require.NoError(t, err)
	require.True(t, password.Check("my-known-pass", user.PasswordHash))
}

func TestEnsureInitialAdmin_Idempotent(t *testing.T) {
	d := db.GetTestDB(t)
	ctx := t.Context()

	created1, pass1, err := ensureInitialAdmin(ctx, d, "admin@example.com", "")
	require.NoError(t, err)
	require.True(t, created1)
	require.NotEmpty(t, pass1)

	// Second call with the same email does nothing.
	created2, pass2, err := ensureInitialAdmin(ctx, d, "admin@example.com", "")
	require.NoError(t, err)
	require.False(t, created2)
	require.Empty(t, pass2)

	// Original password still works.
	user, err := d.QueryRO().GetUserByEmail(ctx, "admin@example.com")
	require.NoError(t, err)
	require.True(t, password.Check(pass1, user.PasswordHash))
}
