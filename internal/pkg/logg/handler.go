package logg

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

// LoggConfig is the logging configuration.
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type LoggConfig struct {
	// LogPretty enables pretty logging.
	// Default is false (JSON logging).
	LogPretty bool `arg:"--log-pretty,env:LOG_PRETTY" default:"false" help:"Log pretty/with colors"`
	// LogLevel is the log level.
	LogLevel slog.Level `arg:"--log-level,env:LOG_LEVEL" default:"INFO" help:"Log level" placeholder:"LEVEL"`
}

// Validate checks for basic configuration errors.
// LoggConfig has sensible defaults so this currently always returns nil.
func (c *LoggConfig) Validate() error {
	return nil
}

// NewHandler creates a new slog handler from the given config and writer.
func NewHandler(c LoggConfig, w io.Writer) slog.Handler {
	if c.LogPretty {
		return tint.NewHandler(w, &tint.Options{
			AddSource:  true,
			Level:      c.LogLevel,
			TimeFormat: time.TimeOnly,
		})
	}

	return slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource: true,
		Level:     c.LogLevel,
	})
}

func New(c LoggConfig) *slog.Logger {
	return slog.New(NewHandler(c, os.Stderr))
}
