// Package app contains configuration relating to the entire application.
package app

import "time"

// AppConfig contains application-wide configuration.
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type AppConfig struct {
	// AppName is the name of the application.
	// Used e.g. as issuer name for tokens.
	AppName string `arg:"--app-name,env:APP_NAME" default:"" help:"Name of the application, used as issuer in tokens"`
	// ExternalBaseURL is the URL at which the application is reachable from the outside.
	ExternalBaseURL string `arg:"--external-base-url,env:EXTERNAL_BASE_URL" default:"" help:"Base URL of the application"`
	// DevelopmentMode enables some development features.
	DevelopmentMode bool `arg:"--dev-mode,env:DEV_MODE" default:"false" help:"Enable development mode"`
	// EmailJWTSecret is the secret used to sign email verification JWTs.
	// Generated on startup if not set.
	EmailJWTSecret string `arg:"--email-jwt-secret,env:EMAIL_JWT_SECRET" help:"Secret to sign email verification JWTs, generated on startup if not set" placeholder:"SECRET"`
	// EmailVerificationExpiry is how long an email verification link remains valid.
	EmailVerificationExpiry time.Duration `arg:"--email-verification-expiry,env:EMAIL_VERIFICATION_EXPIRY" default:"2h" help:"How long email verification links are valid"`
	// TrackColor is the stroke color used for all track preview SVGs.
	// Accepts a CSS hex value (e.g. "#f00", "#rrggbb") or "currentColor".
	TrackColor string `arg:"--track-color,env:TRACK_COLOR" default:"currentColor" help:"Stroke color for track preview SVGs, CSS hex or currentColor"`
}
