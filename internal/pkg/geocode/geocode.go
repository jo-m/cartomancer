// Package geocode provides reverse geocoding via GeoNames data.
package geocode

import (
	"archive/zip"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/geocode/cols"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

const (
	// AllCountriesURL is the download URL for the GeoNames allCountries dataset.
	AllCountriesURL = cols.BaseURL + "/allCountries.zip"

	// allCountriesEntry is the filename inside the zip archive.
	allCountriesEntry = "allCountries.txt"

	// insertBatchSize is the number of rows inserted per batch in a transaction.
	insertBatchSize = 10000
)

// DownloadAllCountries downloads the allCountries.zip file to a temporary file
// and returns its path. The caller is responsible for removing the file.
func DownloadAllCountries(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AllCountriesURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download allCountries.zip: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d downloading allCountries.zip", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "geonames-*.zip")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("close temp file: %w", err)
	}

	return tmp.Name(), nil
}

// ImportAllCountries reads the zip file at zipPath, parses all geoname rows,
// and replaces the geonames table contents atomically.
func ImportAllCountries(ctx context.Context, d *db.DB, zipPath string) (int, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	var dataFile *zip.File
	for _, f := range zr.File {
		if f.Name == allCountriesEntry {
			dataFile = f
			break
		}
	}
	if dataFile == nil {
		return 0, fmt.Errorf("%s not found in zip archive", allCountriesEntry)
	}

	rc, err := dataFile.Open()
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", allCountriesEntry, err)
	}
	defer rc.Close()

	return importFromReader(ctx, d, rc)
}

// importFromReader parses tab-delimited geoname rows from r and inserts them
// into the database, replacing all existing data.
func importFromReader(ctx context.Context, d *db.DB, r io.Reader) (int, error) {
	// First, delete all existing rows.
	err := d.WithTx(ctx, func(tx *db.Queries) error {
		_, txErr := tx.DeleteAllGeonames(ctx)
		return txErr
	})
	if err != nil {
		return 0, fmt.Errorf("delete existing geonames: %w", err)
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	total := 0
	batch := make([]db.InsertGeonameParams, 0, insertBatchSize)

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		return d.WithTx(ctx, func(tx *db.Queries) error {
			for _, p := range batch {
				if txErr := tx.InsertGeoname(ctx, p); txErr != nil {
					return txErr
				}
			}
			return nil
		})
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		p, err := parseLine(line)
		if errors.Is(err, errSkipped) {
			continue
		}
		if err != nil {
			logg.Debug(ctx, "Skipping malformed geonames line", "err", err)
			continue
		}

		batch = append(batch, p)
		if len(batch) >= insertBatchSize {
			if err := flushBatch(); err != nil {
				return total, fmt.Errorf("flush batch at row %d: %w", total, err)
			}
			total += len(batch)
			batch = batch[:0]

			if total%500000 == 0 {
				logg.Info(ctx, "Geonames import progress", "rows", total)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return total, fmt.Errorf("scan: %w", err)
	}

	// Flush remaining rows.
	if err := flushBatch(); err != nil {
		return total, fmt.Errorf("flush final batch: %w", err)
	}
	total += len(batch)

	return total, nil
}

// errSkipped is returned by parseLine when a row should be silently skipped.
var errSkipped = errors.New("skipped")

// parseLine parses a single tab-delimited geonames line into insert params.
// Returns [errSkipped] for rows that should be filtered out (e.g. undersea features).
func parseLine(line string) (db.InsertGeonameParams, error) {
	fields := strings.Split(line, "\t")
	if len(fields) < cols.NumColumns {
		return db.InsertGeonameParams{}, fmt.Errorf("expected %d fields, got %d", cols.NumColumns, len(fields))
	}

	// Skip undersea features (feature class "U") to save space.
	if fields[cols.IdxFeatureClass] == "U" {
		return db.InsertGeonameParams{}, errSkipped
	}

	geonameid, err := strconv.ParseInt(fields[cols.IdxGeonameid], 10, 64)
	if err != nil {
		return db.InsertGeonameParams{}, fmt.Errorf("parse geonameid: %w", err)
	}

	lat, err := strconv.ParseFloat(fields[cols.IdxLatitude], 64)
	if err != nil {
		return db.InsertGeonameParams{}, fmt.Errorf("parse latitude: %w", err)
	}

	lon, err := strconv.ParseFloat(fields[cols.IdxLongitude], 64)
	if err != nil {
		return db.InsertGeonameParams{}, fmt.Errorf("parse longitude: %w", err)
	}

	return db.InsertGeonameParams{
		Geonameid:    geonameid,
		Name:         fields[cols.IdxName],
		Latitude:     lat,
		Longitude:    lon,
		FeatureClass: fields[cols.IdxFeatureClass],
		FeatureCode:  fields[cols.IdxFeatureCode],
		CountryCode:  fields[cols.IdxCountryCode],
		Cc2:          fields[cols.IdxCc2],
		Admin1Code:   fields[cols.IdxAdmin1Code],
		Admin2Code:   fields[cols.IdxAdmin2Code],
		Admin3Code:   fields[cols.IdxAdmin3Code],
		Admin4Code:   fields[cols.IdxAdmin4Code],
	}, nil
}
