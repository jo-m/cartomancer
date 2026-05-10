package app

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/utl"
)

var testJWTSecret = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("T", utl.JWTSecretMinBytes)))

func validConfig() AppConfig {
	return AppConfig{
		InstanceName:            "Test",
		ExternalBaseURL:         "https://example.com",
		EmailJWTSecret:          testJWTSecret,
		EmailVerificationExpiry: 2 * time.Hour,
	}
}

func TestValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c := validConfig()
		require.NoError(t, c.Validate())
	})

	t.Run("empty instance name", func(t *testing.T) {
		c := validConfig()
		c.InstanceName = ""
		require.ErrorContains(t, c.Validate(), "APP_INSTANCE_NAME")
	})

	t.Run("zero expiry", func(t *testing.T) {
		c := validConfig()
		c.EmailVerificationExpiry = 0
		require.ErrorContains(t, c.Validate(), "APP_EMAIL_VERIFICATION_EXPIRY")
	})

	t.Run("invalid base64 email JWT secret", func(t *testing.T) {
		c := validConfig()
		c.EmailJWTSecret = "not-valid-base64!!!"
		require.ErrorContains(t, c.Validate(), "APP_EMAIL_JWT_SECRET")
	})

	t.Run("too short email JWT secret", func(t *testing.T) {
		c := validConfig()
		c.EmailJWTSecret = base64.StdEncoding.EncodeToString([]byte("short"))
		require.ErrorContains(t, c.Validate(), "APP_EMAIL_JWT_SECRET")
	})

	t.Run("empty email JWT secret is allowed", func(t *testing.T) {
		c := validConfig()
		c.EmailJWTSecret = ""
		require.NoError(t, c.Validate())
	})

	t.Run("negative auth rate limit", func(t *testing.T) {
		c := validConfig()
		c.RateLimitAuthRPS = -1
		require.ErrorContains(t, c.Validate(), "APP_RATE_LIMIT_AUTH_RPS")
	})

	t.Run("negative email-send rate limit", func(t *testing.T) {
		c := validConfig()
		c.RateLimitEmailSendRPS = -1
		require.ErrorContains(t, c.Validate(), "APP_RATE_LIMIT_EMAIL_SEND_RPS")
	})

	t.Run("zero rate limits allowed", func(t *testing.T) {
		c := validConfig()
		c.RateLimitAuthRPS = 0
		c.RateLimitEmailSendRPS = 0
		require.NoError(t, c.Validate())
	})
}
