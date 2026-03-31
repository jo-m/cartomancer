// Command gridprint parses an ICON-CH1 grid-constants GRIB2 file and prints
// the latitude, longitude, and altitude (geopotential height) of every grid
// point as tab-separated values.
//
// Usage:
//
//	go run ./internal/cmd/gridprint internal/pkg/grib2/testdata/vert_const.grib2
//	go run ./internal/cmd/gridprint internal/pkg/grib2/testdata/horiz_const.grib2
package main

import (
	"bufio"
	"fmt"
	"os"

	"jo-m.ch/go/cartomancer/internal/pkg/grib2"
)

// GRIB2 parameter codes for the fields we extract.
const (
	clatCategory  = uint8(191)
	clatParameter = uint8(1)
	clonCategory  = uint8(191)
	clonParameter = uint8(2)
	// Geopotential height: discipline 0, category 3, parameter 6.
	geoHDiscipline = uint8(0)
	geoHCategory   = uint8(3)
	geoHParameter  = uint8(6)
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: gridprint <file.grib2>\n")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	msgs, err := grib2.Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}

	var lats, lons, alts []float32
	for _, m := range msgs {
		switch {
		case m.Category == clatCategory && m.Parameter == clatParameter:
			lats = m.Values
		case m.Category == clonCategory && m.Parameter == clonParameter:
			lons = m.Values
		case m.Discipline == geoHDiscipline && m.Category == geoHCategory && m.Parameter == geoHParameter:
			alts = m.Values
		}
	}

	if lats == nil {
		fmt.Fprintf(os.Stderr, "CLAT field not found\n")
		os.Exit(1)
	}
	if lons == nil {
		fmt.Fprintf(os.Stderr, "CLON field not found\n")
		os.Exit(1)
	}
	if alts == nil {
		fmt.Fprintf(os.Stderr, "geopotential height field (disc=0, cat=3, param=6) not found\n")
		os.Exit(1)
	}
	if len(lats) != len(lons) || len(lats) != len(alts) {
		fmt.Fprintf(os.Stderr, "field lengths differ: CLAT=%d CLON=%d altitude=%d\n", len(lats), len(lons), len(alts))
		os.Exit(1)
	}

	w := bufio.NewWriter(os.Stdout)
	fmt.Fprintln(w, "index\tlat\tlon\talt_m")
	for i := range lats {
		fmt.Fprintf(w, "%d\t%.6f\t%.6f\t%.2f\n", i, lats[i], lons[i], alts[i])
	}
	w.Flush()
}
