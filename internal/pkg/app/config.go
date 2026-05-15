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
	DevInitialAdminPassword = "admin123"
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
	// RateLimitAuthRPS is the per-IP token refill rate applied to the authentication
	// endpoints (/sessions/login, /confirm-email), in requests per second.
	// Fractional values are allowed (e.g. 0.1667 for 10 per minute).
	// Zero disables the limit.
	RateLimitAuthRPS float64 `arg:"--app-rate-limit-auth-rps,env:APP_RATE_LIMIT_AUTH_RPS" default:"0.1" help:"Per-IP token refill rate for auth endpoints (requests/second, 0 to disable)" placeholder:"N"`
	// RateLimitAuthBurst is the maximum burst for the auth rate limiter.
	// Zero means auto: max(int(RateLimitAuthRPS), 1).
	RateLimitAuthBurst int `arg:"--app-rate-limit-auth-burst,env:APP_RATE_LIMIT_AUTH_BURST" default:"5" help:"Max burst for auth rate limiter (0 = auto: max(int(rps),1))" placeholder:"N"`
	// RateLimitEmailSendRPS is the per-IP token refill rate applied to endpoints that
	// trigger sending an email (/register), in requests per second.
	// Fractional values are allowed (e.g. 0.1667 for 10 per minute).
	// Zero disables the limit.
	RateLimitEmailSendRPS float64 `arg:"--app-rate-limit-email-send-rps,env:APP_RATE_LIMIT_EMAIL_SEND_RPS" default:"0.1" help:"Per-IP token refill rate for endpoints triggering email sends (requests/second, 0 to disable)" placeholder:"N"`
	// RateLimitEmailSendBurst is the maximum burst for the email-send rate limiter.
	// Zero means auto: max(int(RateLimitEmailSendRPS), 1).
	RateLimitEmailSendBurst int `arg:"--app-rate-limit-email-send-burst,env:APP_RATE_LIMIT_EMAIL_SEND_BURST" default:"1" help:"Max burst for email-send rate limiter (0 = auto: max(int(rps),1))" placeholder:"N"`
	// RateLimitTrustedProxies is the number of trusted reverse proxies in front of the
	// application. 0 means the direct TCP peer is used as the client IP. When > 0, the
	// real client IP is read from the X-Forwarded-For header by skipping that many
	// rightmost entries (which were appended by trusted proxies).
	RateLimitTrustedProxies int `arg:"--app-rate-limit-trusted-proxies,env:APP_RATE_LIMIT_TRUSTED_PROXIES" default:"0" help:"Number of trusted reverse proxies; 0 = use direct connection IP" placeholder:"N"`
	// RateLimitIPv6PrefixLen is the prefix length used to group IPv6 addresses for
	// rate limiting. All addresses within the same prefix share one token bucket.
	// Valid range: 1-128.
	RateLimitIPv6PrefixLen int `arg:"--app-rate-limit-ipv6-prefix-len,env:APP_RATE_LIMIT_IPV6_PREFIX_LEN" default:"64" help:"IPv6 prefix length for grouping addresses into one rate limit bucket (1-128)" placeholder:"N"`
	// RateLimitMaxIPs is the maximum number of distinct keys held simultaneously in
	// each rate limiter map (used by both the per-IP and per-UID limiters). When the
	// cap is reached, new keys are rejected (fail-closed) until the periodic cleanup
	// evicts idle entries. Sized to be large enough to comfortably hold legitimate
	// traffic so that legitimate users are not turned away.
	RateLimitMaxIPs int `arg:"--app-rate-limit-max-ips,env:APP_RATE_LIMIT_MAX_IPS" default:"100000" help:"Max distinct keys tracked per rate limiter (fail-closed when exceeded)" placeholder:"N"`
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
	if c.RateLimitAuthRPS < 0 {
		return errors.New("--app-rate-limit-auth-rps / APP_RATE_LIMIT_AUTH_RPS must be non-negative")
	}
	if c.RateLimitAuthBurst < 0 {
		return errors.New("--app-rate-limit-auth-burst / APP_RATE_LIMIT_AUTH_BURST must be non-negative")
	}
	if c.RateLimitEmailSendRPS < 0 {
		return errors.New("--app-rate-limit-email-send-rps / APP_RATE_LIMIT_EMAIL_SEND_RPS must be non-negative")
	}
	if c.RateLimitEmailSendBurst < 0 {
		return errors.New("--app-rate-limit-email-send-burst / APP_RATE_LIMIT_EMAIL_SEND_BURST must be non-negative")
	}
	if c.RateLimitTrustedProxies < 0 {
		return errors.New("--app-rate-limit-trusted-proxies / APP_RATE_LIMIT_TRUSTED_PROXIES must be non-negative")
	}
	if c.RateLimitIPv6PrefixLen < 1 || c.RateLimitIPv6PrefixLen > 128 {
		return errors.New("--app-rate-limit-ipv6-prefix-len / APP_RATE_LIMIT_IPV6_PREFIX_LEN must be between 1 and 128")
	}
	if c.RateLimitMaxIPs < 1 {
		return errors.New("--app-rate-limit-max-ips / APP_RATE_LIMIT_MAX_IPS must be at least 1")
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
