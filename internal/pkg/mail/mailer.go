package mail

import (
	"context"
	"fmt"
	"goweb/internal/pkg/jobs"
)

type Args struct {
	To string
}

// Kind implements jobs.Args.
func (a Args) Kind() string { return "main.mailer" }

var _ jobs.Args = (*Args)(nil)

type Mailer struct{}

var _ jobs.Job[Args] = (*Mailer)(nil)

// Run implements jobs.Job.
func (c *Mailer) Run(ctx context.Context, args Args) error {
	// TODO: actually implement.
	return fmt.Errorf("failed to send mail to %s", args.To)
}
