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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	collectionURL = "https://data.geo.admin.ch/api/stac/v0.9/collections/ch.meteoschweiz.ogd-forecasting-icon-ch1"
	itemsURL      = collectionURL + "/items?limit=100&sortby=-datetime"

	outDir = "internal/pkg/grib2/testdata"
)

// target describes one file to download.
type target struct {
	variable string // forecast:variable property
	horizon  string // forecast:horizon property (ISO 8601 duration)
	filename string // destination filename within outDir
}

var targets = []target{
	// Horizontal grid constants (needed by ParseGrid).
	// Downloaded from the collection's static assets, not from items.
	{filename: "horiz_const.grib2"},

	// U at t=0: multi-level zonal wind, 16-bit packed values, PDT 1.
	{variable: "U", horizon: "P0DT00H00M00S", filename: "u_0h.grib2"},

	// V at t=0: multi-level meridional wind, 16-bit packed values, PDT 1.
	{variable: "V", horizon: "P0DT00H00M00S", filename: "v_0h.grib2"},

	// TOT_PR at t=10h: instantaneous total precipitation rate, PDT 1.
	{variable: "TOT_PR", horizon: "P0DT10H00M00S", filename: "tot_pr_10h.grib2"},
}

// stacCollection is a minimal STAC Collection for JSON decoding.
type stacCollection struct {
	Assets map[string]struct {
		Href string `json:"href"`
	} `json:"assets"`
}

// stacPage is a minimal STAC ItemCollection for JSON decoding.
type stacPage struct {
	Features []struct {
		Properties struct {
			Variable  string `json:"forecast:variable"`
			Horizon   string `json:"forecast:horizon"`
			Perturbed bool   `json:"forecast:perturbed"`
		} `json:"properties"`
		Assets map[string]struct {
			Href string `json:"href"`
		} `json:"assets"`
	} `json:"features"`
	Links []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
}

func main() {
	ctx := context.Background()

	// Build a lookup: (variable, horizon) → target.
	byKey := make(map[string]*target)
	for i := range targets {
		t := &targets[i]
		if t.variable != "" {
			byKey[t.variable+"|"+t.horizon] = t
		}
	}

	// Download horizontal constants and parameter CSV from collection assets.
	fmt.Println("Fetching collection...")
	coll, err := fetchJSON[stacCollection](ctx, collectionURL)
	must(err)
	horizAsset, ok := coll.Assets["horizontal_constants_icon-ch1-eps.grib2"]
	if !ok {
		panic("horizontal constants asset not found in collection")
	}
	must(downloadFile(ctx, horizAsset.Href, filepath.Join(outDir, "horiz_const.grib2")))
	fmt.Println("  OK horiz_const.grib2")

	csvAsset, ok := coll.Assets["params_icon-ch1-eps.csv"]
	if !ok {
		panic("params CSV asset not found in collection")
	}
	must(downloadFile(ctx, csvAsset.Href, filepath.Join(outDir, "params_icon-ch1-eps.csv")))
	fmt.Println("  OK params_icon-ch1-eps.csv")

	// Paginate through items to find the remaining targets.
	remaining := len(byKey)
	url := itemsURL
	pageNum := 0
	for url != "" && remaining > 0 {
		pageNum++
		fmt.Printf("Fetching items page %d (%d target(s) remaining)...\n", pageNum, remaining)
		page, err := fetchJSON[stacPage](ctx, url)
		must(err)
		fmt.Printf("  page %d: %d items\n", pageNum, len(page.Features))

		for _, item := range page.Features {
			if item.Properties.Perturbed {
				continue
			}
			key := item.Properties.Variable + "|" + item.Properties.Horizon
			t, ok := byKey[key]
			if !ok {
				continue
			}
			dest := filepath.Join(outDir, t.filename)
			if _, err := os.Stat(dest); err == nil {
				fmt.Printf("  skip %s (already exists)\n", t.filename)
				remaining--
				continue
			}
			fmt.Printf("  downloading %s...\n", t.filename)
			for _, asset := range item.Assets {
				must(downloadFile(ctx, asset.Href, dest))
				fmt.Printf("  OK %s\n", t.filename)
				remaining--
				break
			}
		}

		url = ""
		for _, l := range page.Links {
			if l.Rel == "next" {
				url = l.Href
				break
			}
		}
	}

	if remaining > 0 {
		fmt.Fprintf(os.Stderr, "warning: %d targets not found in STAC collection\n", remaining)
		os.Exit(1)
	}
	fmt.Println("Done.")
}

func fetchJSON[T any](ctx context.Context, url string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	var v T
	return v, json.NewDecoder(resp.Body).Decode(&v)
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
