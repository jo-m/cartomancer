package main

import (
	"context"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/jobs"
	"goweb/internal/pkg/logg"
	"time"
)

// TODO: remove this temporary test job.

type MailerArgs struct {
	To   string
	Body string
}

func (a MailerArgs) Kind() string { return "mailer" }

var _ jobs.Args = (*MailerArgs)(nil)

type Mailer struct{}

var _ jobs.Worker[MailerArgs] = (*Mailer)(nil)

func (m *Mailer) Work(ctx context.Context, _ *db.DB, args MailerArgs) error {
	logg.Info(ctx, "Doing work", "args", args)
	time.Sleep(time.Millisecond * 10)
	return nil
}
