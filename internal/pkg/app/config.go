// Package app contains configuration relating to the entire application.
package app

import "time"

// AppConfig contains application-wide configuration.
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type AppConfig struct {
	// InstanceName is the name of this hosted application instance.
	// Used e.g. as issuer name for tokens.
	InstanceName string `arg:"--app-name,env:APP_INSTANCE_NAME" default:"Detour" help:"Name of the application, used as issuer in tokens"`
	// ExternalBaseURL is the URL at which the application is reachable from the outside.
	ExternalBaseURL string `arg:"--app-external-base-url,env:APP_EXTERNAL_BASE_URL" default:"" help:"Base URL of the application"`
	// DevelopmentMode enables some development features.
	DevelopmentMode bool `arg:"--app-dev-mode,env:APP_DEV_MODE" default:"false" help:"Enable development mode"`
	// EmailJWTSecret is the secret used to sign email verification JWTs.
	// Generated on startup if not set.
	EmailJWTSecret string `arg:"--app-email-jwt-secret,env:APP_EMAIL_JWT_SECRET" help:"Secret to sign email verification JWTs, generated on startup if not set" placeholder:"SECRET"`
	// EmailVerificationExpiry is how long an email verification link remains valid.
	EmailVerificationExpiry time.Duration `arg:"--app-email-verification-expiry,env:APP_EMAIL_VERIFICATION_EXPIRY" default:"2h" help:"How long email verification links are valid"`
	// TrackColor is the stroke color used for all track preview SVGs.
	// Accepts a CSS hex value (e.g. "#f00", "#rrggbb") or "currentColor".
	TrackColor string `arg:"--app-track-color,env:APP_TRACK_COLOR" default:"currentColor" help:"Stroke color for track preview SVGs, CSS hex or currentColor"`
}
