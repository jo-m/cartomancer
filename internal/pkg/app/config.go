// Package app contains configuration relating to the entire application.
package app

import (
	"errors"
	"fmt"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/utl"
)

// DevInitialAdminEmail and DevInitialAdminPassword are the credentials used for
// the pre-created admin account in development and demo mode.
const (
	DevInitialAdminEmail    = "admin@example.com"
	DevInitialAdminPassword = "admin"
)

// AppConfig contains application-wide configuration.
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type AppConfig struct {
	// InstanceName is the name of this hosted application instance.
	// Used e.g. as issuer name for tokens.
	InstanceName string `arg:"--app-instance-name,env:APP_INSTANCE_NAME" default:"Cartomancer" help:"Name of the application instance, used as issuer in tokens" placeholder:"NAME"`
	// ExternalBaseURL is the URL at which the application is reachable from the outside.
	// REQUIRED for production deployments.
	ExternalBaseURL string `arg:"--app-external-base-url,env:APP_EXTERNAL_BASE_URL" default:"http://localhost:8080" help:"Base URL of the application, needed for links and emails" placeholder:"URL"`
	// DevelopmentMode enables some development features.
	// On startup an admin user "test@example.org" with password "asdf" is created.
	// DANGEROUS.
	DevelopmentMode bool `arg:"--app-dev-mode,env:APP_DEV_MODE" default:"false" help:"Enable development mode"`
	// EmailJWTSecret is the secret used to sign email verification JWTs.
	// Generated on startup if not set.
	// REQUIRED for production deployments.
	EmailJWTSecret string `arg:"--app-email-jwt-secret,env:APP_EMAIL_JWT_SECRET" help:"Base64-encoded secret (min 512 bits) to sign email verification JWTs, generated on startup if not set" placeholder:"SECRET"`
	// EmailVerificationExpiry is how long an email verification link remains valid.
	EmailVerificationExpiry time.Duration `arg:"--app-email-verification-expiry,env:APP_EMAIL_VERIFICATION_EXPIRY" default:"2h" help:"How long email verification links are valid" placeholder:"DUR"`
	// RegistrationEnabled controls whether new users can self-register via the /register endpoint.
	// Off by default.
	RegistrationEnabled bool `arg:"--app-registration-enabled,env:APP_REGISTRATION_ENABLED" default:"false" help:"Allow new users to self-register"`
	// InitAdminEmail, when set, creates an initial admin account with this email
	// on startup if no user with this email exists yet.
	// A random password is generated, use the setpass subcommand to reset it.
	InitAdminEmail string `arg:"--app-init-admin-email,env:APP_INIT_ADMIN_EMAIL" default:"" help:"Email for the initial admin account created on first startup" placeholder:"EMAIL"`
	// DemoMode enables a read-only demo instance.
	// Users and email verifications are locked via database triggers (only last_login_at,
	// last_active_at, and sessions are allowed to change), and all tracks are deleted
	// every 30 minutes by a periodic job.
	// A .demo suffix is appended to the database path to prevent accidental overwrites.
	// Cannot be active together with DevelopmentMode.
	DemoMode bool `arg:"--app-demo-mode,env:APP_DEMO_MODE" default:"false" help:"Enable demo mode (locks users, periodically deletes tracks, uses .demo DB suffix)"`
}

// Validate checks for basic configuration errors.
func (c *AppConfig) Validate() error {
	if c.InstanceName == "" {
		return errors.New("--app-instance-name / APP_INSTANCE_NAME must not be empty")
	}
	if c.EmailVerificationExpiry <= 0 {
		return errors.New("--app-email-verification-expiry / APP_EMAIL_VERIFICATION_EXPIRY must be positive")
	}
	if c.EmailJWTSecret != "" {
		if _, err := utl.DecodeJWTSecret(c.EmailJWTSecret); err != nil {
			return fmt.Errorf("--app-email-jwt-secret / APP_EMAIL_JWT_SECRET: %w", err)
		}
	}
	if c.DemoMode && c.DevelopmentMode {
		return errors.New("--app-demo-mode / APP_DEMO_MODE and --app-dev-mode / APP_DEV_MODE cannot both be enabled")
	}
	return nil
}

// ValidateProduction checks that all settings required for a production deployment are set.
func (c *AppConfig) ValidateProduction() error {
	if c.ExternalBaseURL == "" {
		return errors.New("--app-external-base-url / APP_EXTERNAL_BASE_URL is required for production")
	}
	if c.EmailJWTSecret == "" {
		return errors.New("--app-email-jwt-secret / APP_EMAIL_JWT_SECRET is required for production")
	}
	return nil
}
