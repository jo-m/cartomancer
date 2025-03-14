package logg

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

// Config is the logging configuration. It contains struct tags compatible with github.com/alexflint/go-arg.
type Config struct {
	LogPretty bool       `arg:"--log-pretty,env:LOG_PRETTY" default:"false" help:"Log pretty/with colors"`
	LogLevel  slog.Level `arg:"--log-level,env:LOG_LEVEL" default:"INFO" help:"Log level" placeholder:"LEVEL"`
}

func NewHandler(c Config) slog.Handler {
	if c.LogPretty {
		return tint.NewHandler(os.Stderr, &tint.Options{
			// TODO: Skip levels!
			AddSource:  true,
			Level:      c.LogLevel,
			TimeFormat: time.Kitchen,
		})
	}

	return slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     c.LogLevel,
	})
}
