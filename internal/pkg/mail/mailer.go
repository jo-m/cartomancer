// Package mail provides a job that sends emails.
package mail

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jo-m/goweb/internal/pkg/jobs"
	"github.com/jo-m/goweb/internal/pkg/logg"
	"github.com/wneessen/go-mail"
	"github.com/wneessen/go-mail/log"
)

// MailerConfig is the configuration for a mailer.
// It has struct tags compatible with github.com/alexflint/go-arg.
//
//revive:disable:exported Naming necessary for struct embedding.
type MailerConfig struct {
	SMTPHost  string `arg:"--mail-smtp-host,env:MAIL_SMTP_HOST" default:"localhost" help:"SMTP server host" placeholder:"HOST"`
	SMTPPort  uint16 `arg:"--mail-smtp-port,env:MAIL_SMTP_PORT" default:"25" help:"SMTP server port" placeholder:"PORT"`
	SMTPNoTLS bool   `arg:"--mail-smtp-no-tls,env:MAIL_SMTP_NO_TLS" default:"false" help:"Do not require TLS for SMTP" placeholder:"BOOL"`

	AuthUsername string `arg:"--mail-auth-username,env:MAIL_AUTH_USERNAME" default:"" help:"SMTP auth username" placeholder:"USER"`
	AuthPassword string `arg:"--mail-auth-password,env:MAIL_AUTH_PASSWORD" default:"" help:"SMTP auth password" placeholder:"PASS"`

	From string `arg:"--mail-from,env:MAIL_FROM" default:"" help:"Sender email address" placeholder:"EMAIL"`
}

type Args struct {
	To      []string
	Subject string
	Body    string
}

// Kind implements jobs.Args.
func (a Args) Kind() string { return "main.mailer" }

var _ jobs.Args = (*Args)(nil)

// Mailer is a job that sends emails.
// Use NewMailer to create a new instance.
type Mailer struct {
	c MailerConfig
}

// NewMailer creates a new Mailer.
func NewMailer(c MailerConfig) *Mailer {
	return &Mailer{c: c}
}

var _ jobs.Job[Args] = (*Mailer)(nil)

type mailLogger struct {
	l *slog.Logger
}

var _ log.Logger = (*mailLogger)(nil)

func (l *mailLogger) Debugf(msg log.Log) {
	l.l.Debug(fmt.Sprintf(msg.Format, msg.Messages...), "dir", msg.Direction)
}

func (l *mailLogger) Infof(msg log.Log) {
	l.l.Info(fmt.Sprintf(msg.Format, msg.Messages...), "dir", msg.Direction)
}

func (l *mailLogger) Warnf(msg log.Log) {
	l.l.Warn(fmt.Sprintf(msg.Format, msg.Messages...), "dir", msg.Direction)
}

func (l *mailLogger) Errorf(msg log.Log) {
	l.l.Error(fmt.Sprintf(msg.Format, msg.Messages...), "dir", msg.Direction)
}

func (m *Mailer) configureClient(logger *slog.Logger) (*mail.Client, error) {
	opts := []mail.Option{
		mail.WithDSN(),
		mail.WithLogger(&mailLogger{l: logger}),
		mail.WithPort(int(m.c.SMTPPort)),
	}
	if m.c.AuthUsername != "" || m.c.AuthPassword != "" {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover))
		opts = append(opts, mail.WithUsername(m.c.AuthUsername))
		opts = append(opts, mail.WithPassword(m.c.AuthPassword))
	}
	if m.c.SMTPNoTLS {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSOpportunistic))
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	}

	return mail.NewClient(m.c.SMTPHost, opts...)
}

// Run implements jobs.Job.
func (m *Mailer) Run(ctx context.Context, args Args) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	client, err := m.configureClient(logg.GetLogger(ctx))
	if err != nil {
		return fmt.Errorf("failed to configure SMTP client: %w", err)
	}
	defer client.Close()

	msg := mail.NewMsg()
	if err := msg.From(m.c.From); err != nil {
		return fmt.Errorf("failed to set From address: %s", err)
	}
	if err := msg.To(args.To...); err != nil {
		return fmt.Errorf("failed to set To addresses: %s", err)
	}
	msg.Subject(args.Subject)
	msg.SetBodyString(mail.TypeTextPlain, args.Body)

	return client.DialAndSend(msg)
}
