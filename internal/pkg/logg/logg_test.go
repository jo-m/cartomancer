package logg

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
(from https://github.com/acarl005/stripansi/blob/master/stripansi.go)

# MIT License

# Copyright (c) 2018 Andrew Carlson

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
var stripAnsiRe = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func stripAnsi(str string) string {
	return stripAnsiRe.ReplaceAllString(str, "")
}

func TestLoggDisableDefaultLogger(t *testing.T) {
	slog.Info("hello")
	DisableDefaultLogger()
	assert.PanicsWithError(t, ErrNotAllowed.Error(), func() {
		slog.Info("hello")
	})
}

func TestLoggContext(t *testing.T) {
	logger := New(LoggConfig{})
	ctx := WithLogger(t.Context(), logger)
	require.NotNil(t, ctx)

	logger2 := GetLogger(ctx)
	assert.Equal(t, logger, logger2)

	assert.Equal(t, GetLogger(t.Context()), slog.Default())
}

func TestLoggPanic(t *testing.T) {
	logger := New(LoggConfig{})
	ctx := WithLogger(t.Context(), logger)

	assert.Panics(t, func() {
		Panic(ctx, "crash")
	})
}

func testLog(ctx context.Context, logger *slog.Logger) {
	slog.Info("hello", "from", "slog", "via", "slog.Info")
	slog.Log(ctx, slog.LevelWarn, "hello", "from", "slog", "via", "slog.Log")

	logger.Info("hello", "logger", "slog", "via", "logger.Info")
	logger.Log(ctx, slog.LevelWarn, "hello", "logger", "slog", "via", "logger.Log")

	Info(context.Background(), "hello", "logger", "logg", "via", "Info", "no", "logger")
	Log(context.Background(), slog.LevelWarn, "hello", "logger", "logg", "via", "Log", "no", "logger")

	Trace(ctx, "hello", "logger", "logg", "via", "Trace")
	Debug(ctx, "hello", "logger", "logg", "via", "Debug")
	Info(ctx, "hello", "logger", "logg", "via", "Info")
	Log(ctx, slog.LevelWarn, "hello", "logger", "logg", "via", "Log")
	Error(ctx, "hello", "logger", "logg", "via", "Error")
	Err(ctx, "hello", fmt.Errorf("error"), "logger", "logg", "via", "Err")
	Panic(ctx, "hello", "logger", "logg", "via", "Panic")
}

var (
	// 2025-03-27T00:11:22.923660223
	timeLongRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.[^"]+`)
	// 11:22:17
	timeShortRe = regexp.MustCompile(`\d{2}:\d{2}:\d{2}`)
	// .go:89
	prettyLineNoRe = regexp.MustCompile(`\.go:\d{1,} `)
	// "line":89
	jsonLineNoRe = regexp.MustCompile(`"line":\d{1,}`)
)

func replaceTime(s string) string {
	return timeLongRe.ReplaceAllString(timeShortRe.ReplaceAllString(s, "00:11:22"), "0000-11-22T33:44:55.666Z")
}

func replaceCwd(t *testing.T, s string) string {
	dir, err := os.Getwd()
	require.NoError(t, err)
	return regexp.MustCompile(regexp.QuoteMeta(dir)).ReplaceAllString(s, "")
}

func replaceLineNo(s string) string {
	return prettyLineNoRe.ReplaceAllString(jsonLineNoRe.ReplaceAllString(s, `"line":0`), ".go:0 ")
}

func TestLoggLoggingJSON(t *testing.T) {
	buf := bytes.Buffer{}
	conf := LoggConfig{LogPretty: false}
	logger := slog.New(NewHandler(conf, &buf))
	slog.SetDefault(logger)
	ctx := WithLogger(t.Context(), logger)

	assert.Panics(t, func() {
		testLog(ctx, logger)
	})

	deterministic := replaceLineNo(replaceCwd(t, replaceTime(buf.String())))
	assert.Equal(t, `{"time":"0000-11-22T33:44:55.666Z","level":"INFO","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","from":"slog","via":"slog.Info"}
{"time":"0000-11-22T33:44:55.666Z","level":"WARN","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","from":"slog","via":"slog.Log"}
{"time":"0000-11-22T33:44:55.666Z","level":"INFO","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","logger":"slog","via":"logger.Info"}
{"time":"0000-11-22T33:44:55.666Z","level":"WARN","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","logger":"slog","via":"logger.Log"}
{"time":"0000-11-22T33:44:55.666Z","level":"INFO","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","logger":"logg","via":"Info","no":"logger"}
{"time":"0000-11-22T33:44:55.666Z","level":"WARN","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","logger":"logg","via":"Log","no":"logger"}
{"time":"0000-11-22T33:44:55.666Z","level":"INFO","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","logger":"logg","via":"Info"}
{"time":"0000-11-22T33:44:55.666Z","level":"WARN","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","logger":"logg","via":"Log"}
{"time":"0000-11-22T33:44:55.666Z","level":"ERROR","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","logger":"logg","via":"Error"}
{"time":"0000-11-22T33:44:55.666Z","level":"ERROR","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","logger":"logg","via":"Err","err":"error"}
{"time":"0000-11-22T33:44:55.666Z","level":"ERROR+2","source":{"function":"goweb/internal/pkg/logg.testLog","file":"/logg_test.go","line":0},"msg":"hello","logger":"logg","via":"Panic"}
`, deterministic)
}

func TestLoggLoggingPretty(t *testing.T) {
	buf := bytes.Buffer{}
	conf := LoggConfig{LogPretty: true}
	logger := slog.New(NewHandler(conf, &buf))
	slog.SetDefault(logger)
	ctx := WithLogger(t.Context(), logger)

	assert.Panics(t, func() {
		testLog(ctx, logger)
	})

	deterministic := replaceLineNo(replaceTime(stripAnsi(buf.String())))
	assert.Equal(t, `00:11:22 INF logg/logg_test.go:0 hello from=slog via=slog.Info
00:11:22 WRN logg/logg_test.go:0 hello from=slog via=slog.Log
00:11:22 INF logg/logg_test.go:0 hello logger=slog via=logger.Info
00:11:22 WRN logg/logg_test.go:0 hello logger=slog via=logger.Log
00:11:22 INF logg/logg_test.go:0 hello logger=logg via=Info no=logger
00:11:22 WRN logg/logg_test.go:0 hello logger=logg via=Log no=logger
00:11:22 INF logg/logg_test.go:0 hello logger=logg via=Info
00:11:22 WRN logg/logg_test.go:0 hello logger=logg via=Log
00:11:22 ERR logg/logg_test.go:0 hello logger=logg via=Error
00:11:22 ERR logg/logg_test.go:0 hello logger=logg via=Err err=error
00:11:22 ERR+2 logg/logg_test.go:0 hello logger=logg via=Panic
`, deterministic)
}
