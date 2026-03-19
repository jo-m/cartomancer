package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func validConfig() AppConfig {
	return AppConfig{
		InstanceName:            "Test",
		ExternalBaseURL:         "https://example.com",
		EmailJWTSecret:          "secret",
		EmailVerificationExpiry: 2 * time.Hour,
		TrackColor:              "currentColor",
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

	t.Run("empty track color", func(t *testing.T) {
		c := validConfig()
		c.TrackColor = ""
		require.ErrorContains(t, c.Validate(), "APP_TRACK_COLOR")
	})
}
