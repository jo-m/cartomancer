// Package geonames provides reverse geocoding via GeoNames data.
package geonames

import (
	"archive/zip"
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/geonames/cols"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

// DataAttribution is the TASL attribution for GeoNames geographical data.
// Verified by TestOnlineGeoNamesLicense.
var DataAttribution = attribute.Attribution{
	What:       "Reverse Geocoding",
	Title:      "GeoNames Geographical Database",
	Author:     "GeoNames",
	Source:     "https://www.geonames.org/",
	License:    "CC BY 4.0",
	LicenseURL: "https://creativecommons.org/licenses/by/4.0/",
}

const (
	// AllCountriesURL is the download URL for the GeoNames allCountries dataset.
	AllCountriesURL = cols.BaseURL + "/allCountries.zip"

	// allCountriesEntry is the filename inside the zip archive.
	allCountriesEntry = "allCountries.txt"

	// insertBatchSize is the number of rows per transaction during staging import.
	insertBatchSize = 50000

	// multiInsertRows is the number of rows per multi-row INSERT statement.
	// With 14 columns this uses 14*500 = 7000 parameters, well within SQLite limits.
	multiInsertRows = 500

	// geonamesCols is the number of columns in the geonames table.
	geonamesCols = 14
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

// stagingTable is the name of the temporary staging table used during import.
const stagingTable = "geonames_staging"

// importFromReader parses tab-delimited geoname rows from r and inserts them
// into a staging table, then atomically swaps it with the live geonames table.
// This approach keeps the old data queryable throughout the import and avoids
// long-held write locks by deferring index creation until after all inserts.
func importFromReader(ctx context.Context, d *db.DB, r io.Reader) (int, error) {
	rw := d.RW()

	// Prepare the staging table (unindexed for fast inserts).
	if err := createStagingTable(ctx, rw); err != nil {
		return 0, err
	}

	// On failure, clean up the staging table.
	committed := false
	defer func() {
		if !committed {
			_, _ = rw.ExecContext(ctx, "DROP TABLE IF EXISTS "+stagingTable)
		}
	}()

	// Parse and insert all rows into the staging table.
	total, err := insertIntoStaging(ctx, rw, r)
	if err != nil {
		return total, err
	}

	// Atomically swap the staging table into place.
	if err := swapStagingTable(ctx, rw); err != nil {
		return total, fmt.Errorf("swap staging table: %w", err)
	}
	committed = true

	// Build indexes on the now-live table. Queries work during this time but
	// may be slower until indexes are ready. This avoids index name conflicts
	// between the old and staging tables, and keeps the swap instant.
	if err := createGeonamesIndexes(ctx, rw); err != nil {
		return total, fmt.Errorf("create indexes: %w", err)
	}

	return total, nil
}

// createStagingTable creates the staging table, dropping any leftover from a previous failed import.
func createStagingTable(ctx context.Context, rw *sql.DB) error {
	_, err := rw.ExecContext(ctx, "DROP TABLE IF EXISTS "+stagingTable)
	if err != nil {
		return fmt.Errorf("drop old staging table: %w", err)
	}

	_, err = rw.ExecContext(ctx, `CREATE TABLE `+stagingTable+` (
		geonameid INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		asciiname TEXT NOT NULL DEFAULT '',
		latitude REAL NOT NULL,
		longitude REAL NOT NULL,
		feature_class TEXT NOT NULL DEFAULT '',
		feature_code TEXT NOT NULL DEFAULT '',
		country_code TEXT NOT NULL DEFAULT '',
		cc2 TEXT NOT NULL DEFAULT '',
		admin1_code TEXT NOT NULL DEFAULT '',
		admin2_code TEXT NOT NULL DEFAULT '',
		admin3_code TEXT NOT NULL DEFAULT '',
		admin4_code TEXT NOT NULL DEFAULT '',
		population INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return fmt.Errorf("create staging table: %w", err)
	}
	return nil
}

// createGeonamesIndexes builds the indexes on the geonames table.
// Called after the staging table has been renamed to geonames.
func createGeonamesIndexes(ctx context.Context, rw *sql.DB) error {
	for _, ddl := range []string{
		`CREATE INDEX IF NOT EXISTS idx_geonames_reverse ON geonames (feature_class, latitude, longitude)`,
		`CREATE INDEX IF NOT EXISTS idx_geonames_country ON geonames (country_code, feature_class)`,
	} {
		if _, err := rw.ExecContext(ctx, ddl); err != nil {
			return err
		}
	}

	if err := rebuildGeonamesFTS(ctx, rw); err != nil {
		return fmt.Errorf("rebuild FTS index: %w", err)
	}

	return nil
}

// rebuildGeonamesFTS recreates the FTS5 virtual table and populates it from
// the geonames table. The table is dropped and recreated (rather than using
// 'rebuild') because the content table was swapped via DROP+RENAME.
func rebuildGeonamesFTS(ctx context.Context, rw *sql.DB) error {
	logg.Info(ctx, "rebuilding geonames FTS index")

	for _, stmt := range []string{
		`DROP TABLE IF EXISTS geonames_fts`,
		`CREATE VIRTUAL TABLE geonames_fts USING fts5(
			name, asciiname,
			content=geonames, content_rowid=geonameid,
			tokenize='unicode61 remove_diacritics 2'
		)`,
		`INSERT INTO geonames_fts(geonames_fts) VALUES('rebuild')`,
	} {
		if _, err := rw.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:min(len(stmt), 60)], err)
		}
	}

	logg.Info(ctx, "geonames FTS index rebuilt")
	return nil
}

// swapStagingTable atomically replaces the live geonames table with the staging table.
func swapStagingTable(ctx context.Context, rw *sql.DB) error {
	tx, err := rw.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin swap tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS geonames"); err != nil {
		return fmt.Errorf("drop live table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "ALTER TABLE "+stagingTable+" RENAME TO geonames"); err != nil {
		return fmt.Errorf("rename staging table: %w", err)
	}
	return tx.Commit()
}

// insertIntoStaging parses rows from r and bulk-inserts them into the staging table.
func insertIntoStaging(ctx context.Context, rw *sql.DB, r io.Reader) (int, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	total := 0
	batch := make([]db.InsertGeonameParams, 0, insertBatchSize)

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		return flushToStaging(ctx, rw, batch)
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
			logg.Debug(ctx, "skipping malformed geonames line", "err", err)
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
				logg.Info(ctx, "geonames import progress", "rows", total)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return total, fmt.Errorf("scan: %w", err)
	}

	if err := flushBatch(); err != nil {
		return total, fmt.Errorf("flush final batch: %w", err)
	}
	total += len(batch)

	return total, nil
}

// flushToStaging inserts a batch of rows into the staging table using multi-row
// INSERT statements within a single transaction.
func flushToStaging(ctx context.Context, rw *sql.DB, batch []db.InsertGeonameParams) error {
	tx, err := rw.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for i := 0; i < len(batch); i += multiInsertRows {
		end := min(i+multiInsertRows, len(batch))
		chunk := batch[i:end]

		if err := insertChunk(ctx, tx, chunk); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// insertChunk inserts a chunk of rows using a single multi-row INSERT statement.
func insertChunk(ctx context.Context, tx *sql.Tx, chunk []db.InsertGeonameParams) error {
	var sb strings.Builder
	sb.WriteString("INSERT INTO " + stagingTable + " (geonameid, name, asciiname, latitude, longitude, feature_class, feature_code, country_code, cc2, admin1_code, admin2_code, admin3_code, admin4_code, population) VALUES ")

	args := make([]any, 0, len(chunk)*geonamesCols)
	for i, p := range chunk {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("(?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
		args = append(args, p.Geonameid, p.Name, p.Asciiname, p.Latitude, p.Longitude,
			p.FeatureClass, p.FeatureCode, p.CountryCode, p.Cc2,
			p.Admin1Code, p.Admin2Code, p.Admin3Code, p.Admin4Code, p.Population)
	}

	_, err := tx.ExecContext(ctx, sb.String(), args...)
	return err
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

	var population int64
	if fields[cols.IdxPopulation] != "" {
		population, err = strconv.ParseInt(fields[cols.IdxPopulation], 10, 64)
		if err != nil {
			return db.InsertGeonameParams{}, fmt.Errorf("parse population: %w", err)
		}
	}

	return db.InsertGeonameParams{
		Geonameid:    geonameid,
		Name:         fields[cols.IdxName],
		Asciiname:    fields[cols.IdxAsciiname],
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
		Population:   population,
	}, nil
}
