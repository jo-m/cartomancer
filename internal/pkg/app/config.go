// Package app contains configuration relating to the entire application.
package app

import (
	"errors"
	"time"
)

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
	InstanceName string `arg:"--app-instance-name,env:APP_INSTANCE_NAME" default:"Detour" help:"Name of the application instance, used as issuer in tokens" placeholder:"NAME"`
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
	EmailJWTSecret string `arg:"--app-email-jwt-secret,env:APP_EMAIL_JWT_SECRET" help:"Secret to sign email verification JWTs, generated on startup if not set" placeholder:"SECRET"`
	// EmailVerificationExpiry is how long an email verification link remains valid.
	EmailVerificationExpiry time.Duration `arg:"--app-email-verification-expiry,env:APP_EMAIL_VERIFICATION_EXPIRY" default:"2h" help:"How long email verification links are valid" placeholder:"DUR"`
	// RegistrationEnabled controls whether new users can self-register via the /register endpoint.
	// Off by default.
	RegistrationEnabled bool `arg:"--app-registration-enabled,env:APP_REGISTRATION_ENABLED" default:"false" help:"Allow new users to self-register"`
	// InitAdminEmail, when set, creates an initial admin account with this email
	// on startup if no user with this email exists yet.
	// A random password is generated and logged once.
	InitAdminEmail string `arg:"--app-init-admin-email,env:APP_INIT_ADMIN_EMAIL" default:"" help:"Email for the initial admin account created on first startup" placeholder:"EMAIL"`
	// TrackColor is the stroke color used for all track preview SVGs.
	// Accepts a CSS hex value (e.g. "#f00", "#rrggbb") or "currentColor".
	TrackColor string `arg:"--app-track-color,env:APP_TRACK_COLOR" default:"currentColor" help:"Stroke color for track preview SVGs, CSS hex or currentColor" placeholder:"COLOR"`
}

// Validate checks for basic configuration errors.
func (c *AppConfig) Validate() error {
	if c.InstanceName == "" {
		return errors.New("--app-instance-name / APP_INSTANCE_NAME must not be empty")
	}
	if c.EmailVerificationExpiry <= 0 {
		return errors.New("--app-email-verification-expiry / APP_EMAIL_VERIFICATION_EXPIRY must be positive")
	}
	if c.TrackColor == "" {
		return errors.New("--app-track-color / APP_TRACK_COLOR must not be empty")
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
