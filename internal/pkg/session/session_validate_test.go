package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func validSessionConfig() SessionConfig {
	return SessionConfig{
		IdleTimeout:     24 * time.Hour,
		AbsoluteTimeout: 72 * time.Hour,
		CookieName:      "sid",
		JWTSecret:       "supersecret",
		CookieDomain:    "example.com",
		CookiePath:      "/",
	}
}

func TestSessionConfigValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c := validSessionConfig()
		require.NoError(t, c.Validate())
	})

	t.Run("zero idle timeout", func(t *testing.T) {
		c := validSessionConfig()
		c.IdleTimeout = 0
		require.ErrorContains(t, c.Validate(), "SESSION_IDLE_TIMEOUT")
	})

	t.Run("zero absolute timeout", func(t *testing.T) {
		c := validSessionConfig()
		c.AbsoluteTimeout = 0
		require.ErrorContains(t, c.Validate(), "SESSION_ABS_TIMEOUT")
	})

	t.Run("idle exceeds absolute", func(t *testing.T) {
		c := validSessionConfig()
		c.IdleTimeout = 100 * time.Hour
		require.ErrorContains(t, c.Validate(), "SESSION_IDLE_TIMEOUT")
	})

	t.Run("empty cookie name", func(t *testing.T) {
		c := validSessionConfig()
		c.CookieName = ""
		require.ErrorContains(t, c.Validate(), "SESSION_COOKIE_NAME")
	})
}

func TestSessionConfigValidateProduction(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c := validSessionConfig()
		require.NoError(t, c.ValidateProduction())
	})

	t.Run("missing JWT secret", func(t *testing.T) {
		c := validSessionConfig()
		c.JWTSecret = ""
		require.ErrorContains(t, c.ValidateProduction(), "SESSION_JWT_SECRET")
	})

	t.Run("missing cookie domain", func(t *testing.T) {
		c := validSessionConfig()
		c.CookieDomain = ""
		require.ErrorContains(t, c.ValidateProduction(), "SESSION_COOKIE_DOMAIN")
	})
}
