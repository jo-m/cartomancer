package mail

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMailerConfig_Enabled(t *testing.T) {
	tests := []struct {
		name string
		c    MailerConfig
		want bool
	}{
		{
			name: "all set",
			c:    MailerConfig{SMTPHost: "smtp.example.com", SMTPPort: 587, From: "noreply@example.com"},
			want: true,
		},
		{
			name: "missing host",
			c:    MailerConfig{SMTPHost: "", SMTPPort: 587, From: "noreply@example.com"},
			want: false,
		},
		{
			name: "missing port",
			c:    MailerConfig{SMTPHost: "smtp.example.com", SMTPPort: 0, From: "noreply@example.com"},
			want: false,
		},
		{
			name: "missing from",
			c:    MailerConfig{SMTPHost: "smtp.example.com", SMTPPort: 587, From: ""},
			want: false,
		},
		{
			name: "all empty",
			c:    MailerConfig{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.c.Enabled())
		})
	}
}

func TestMailerConfig_Validate(t *testing.T) {
	c := MailerConfig{}
	require.NoError(t, c.Validate(), "Validate should always succeed")
}
