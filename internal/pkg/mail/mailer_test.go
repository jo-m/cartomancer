package mail

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validMailerConfig() MailerConfig {
	return MailerConfig{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		From:     "noreply@example.com",
	}
}

func TestMailerConfigValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c := validMailerConfig()
		require.NoError(t, c.Validate())
	})

	t.Run("empty SMTP host", func(t *testing.T) {
		c := validMailerConfig()
		c.SMTPHost = ""
		require.ErrorContains(t, c.Validate(), "MAIL_SMTP_HOST")
	})

	t.Run("zero SMTP port", func(t *testing.T) {
		c := validMailerConfig()
		c.SMTPPort = 0
		require.ErrorContains(t, c.Validate(), "MAIL_SMTP_PORT")
	})
}

func TestMailerConfigValidateProduction(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c := validMailerConfig()
		require.NoError(t, c.ValidateProduction())
	})

	t.Run("missing from address", func(t *testing.T) {
		c := validMailerConfig()
		c.From = ""
		require.ErrorContains(t, c.ValidateProduction(), "MAIL_FROM")
	})

	t.Run("TLS disabled", func(t *testing.T) {
		c := validMailerConfig()
		c.SMTPNoTLS = true
		require.ErrorContains(t, c.ValidateProduction(), "MAIL_SMTP_NO_TLS")
	})
}
