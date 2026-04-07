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
}
