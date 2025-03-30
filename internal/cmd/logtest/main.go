package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jo-m/goweb/internal/pkg/logg"
)

// go run ./internal/cmd/logtest/main.go
func main() {
	ctx := context.Background()

	// Initialize logging.
	logg.DisableDefaultLogger()
	conf := logg.LoggConfig{
		LogLevel:  logg.LevelTrace,
		LogPretty: true,
	}
	logger := logg.New(conf)
	slog.SetDefault(logger)
	ctx = logg.WithLogger(ctx, logger)

	slog.Info("hello", "from", "slog", "via", "slog.Info")
	slog.Log(ctx, slog.LevelWarn, "hello", "from", "slog", "via", "slog.Log")

	logger.Info("hello", "logger", "slog", "via", "logger.Info")
	logger.Log(ctx, slog.LevelWarn, "hello", "logger", "slog", "via", "logger.Log")

	logg.Info(context.Background(), "hello", "logger", "logg", "via", "Info", "no", "logger")
	logg.Log(context.Background(), slog.LevelWarn, "hello", "logger", "logg", "via", "Log", "no", "logger")

	logg.Trace(ctx, "hello", "logger", "logg", "via", "logg.Trace")
	logg.Debug(ctx, "hello", "logger", "logg", "via", "logg.Debug")
	logg.Info(ctx, "hello", "logger", "logg", "via", "logg.Info")
	logg.Log(ctx, slog.LevelWarn, "hello", "logger", "logg", "via", "logg.Log")
	logg.Error(ctx, "hello", "logger", "logg", "via", "logg.Error")
	logg.Err(ctx, "hello", errors.New("my error"), "logger", "logg", "via", "logg.Err")
	logg.Panic(ctx, "hello", "logger", "logg", "via", "logg.Panic")
}
