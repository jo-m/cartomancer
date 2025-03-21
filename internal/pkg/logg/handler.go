package logg

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

// LoggConfig is the logging configuration.
// It has struct tags compatible with github.com/alexflint/go-arg.
//
//revive:disable:exported Naming necessary for struct embedding.
type LoggConfig struct {
	// LogPretty enables pretty logging.
	// Default is false (JSON logging).
	LogPretty bool `arg:"--log-pretty,env:LOG_PRETTY" default:"false" help:"Log pretty/with colors"`
	// LogLevel is the log level.
	LogLevel slog.Level `arg:"--log-level,env:LOG_LEVEL" default:"INFO" help:"Log level" placeholder:"LEVEL"`
}

func NewHandler(c LoggConfig) slog.Handler {
	if c.LogPretty {
		return tint.NewHandler(os.Stderr, &tint.Options{
			// TODO: Skip levels!
			// https://github.com/golang/go/issues/59145#issuecomment-1481920720
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
