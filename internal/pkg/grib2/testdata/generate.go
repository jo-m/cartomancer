//go:build ignore

// generate downloads a representative set of GRIB2 forecast files from the
// MeteoSwiss STAC API and writes them to this directory.
//
// STAC Browser: https://data.geo.admin.ch/browser/index.html#/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1
//
// Run from the repository root:
//
//	go run ./internal/pkg/grib2/testdata/generate.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"jo-m.ch/go/detour/internal/pkg/forecast"
	"jo-m.ch/go/detour/internal/pkg/forecast/vars"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

const outDir = "internal/pkg/grib2/testdata"

func main() {
	logger := logg.New(logg.LoggConfig{LogPretty: true, LogLevel: logg.LevelTrace})
	slog.SetDefault(logger)

	result, err := forecast.Download(context.Background(), []vars.Variable{vars.VarT2m, vars.VarTotPr}, 0, false)
	must(err)
	defer os.RemoveAll(result.Dir)

	must(utl.CopyFile(filepath.Join(result.Dir, result.GridConstantsPath), filepath.Join(outDir, "horiz_const.grib2")))
	fmt.Println("  OK horiz_const.grib2")

	if len(result.Files) == 0 {
		panic("no files")
	}
	for _, f := range result.Files {
		hours := int(f.Meta.Horizon.Hours())
		name := fmt.Sprintf("%s_%dh.grib2", strings.ToLower(f.Meta.Variable), hours)
		dest := filepath.Join(outDir, name)
		must(utl.CopyFile(filepath.Join(result.Dir, f.Path), dest))
		fmt.Printf("  OK %s\n", name)
	}

	fmt.Println("Done.")
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
