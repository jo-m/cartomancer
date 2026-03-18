package geocode

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/geocode/cols"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

const (
	// Admin1CodesURL is the download URL for the GeoNames admin1 codes file.
	Admin1CodesURL = cols.BaseURL + "/admin1CodesASCII.txt"

	// Admin2CodesURL is the download URL for the GeoNames admin2 codes file.
	Admin2CodesURL = cols.BaseURL + "/admin2Codes.txt"

	// adminCodeColumns is the expected number of tab-delimited columns in admin code files.
	adminCodeColumns = 4
)

// adminCodeRow holds a parsed row from admin1CodesASCII.txt or admin2Codes.txt.
type adminCodeRow struct {
	Code      string
	Name      string
	Geonameid int64
}

// DownloadAdminCodes downloads the admin1 and admin2 code files and returns
// their contents. Returns (admin1Data, admin2Data, error).
func DownloadAdminCodes() ([]byte, []byte, error) {
	admin1, err := utl.DownloadFile(Admin1CodesURL)
	if err != nil {
		return nil, nil, fmt.Errorf("download admin1 codes: %w", err)
	}

	admin2, err := utl.DownloadFile(Admin2CodesURL)
	if err != nil {
		return nil, nil, fmt.Errorf("download admin2 codes: %w", err)
	}

	return admin1, admin2, nil
}

// ImportAdmin1Codes reads admin1 code rows from r and replaces the
// geoname_admin1 table contents.
func ImportAdmin1Codes(ctx context.Context, d *db.DB, r io.Reader) (int, error) {
	rows, err := parseAdminCodes(r)
	if err != nil {
		return 0, fmt.Errorf("parse admin1 codes: %w", err)
	}

	err = d.WithTx(ctx, func(tx *db.Queries) error {
		if _, txErr := tx.DeleteAllGeonameAdmin1(ctx); txErr != nil {
			return txErr
		}
		for _, row := range rows {
			if txErr := tx.InsertGeonameAdmin1(ctx, db.InsertGeonameAdmin1Params{
				Code:      row.Code,
				Name:      row.Name,
				Geonameid: row.Geonameid,
			}); txErr != nil {
				return txErr
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("import admin1 codes: %w", err)
	}

	logg.Info(ctx, "Admin1 codes imported", "rows", len(rows))
	return len(rows), nil
}

// ImportAdmin2Codes reads admin2 code rows from r and replaces the
// geoname_admin2 table contents.
func ImportAdmin2Codes(ctx context.Context, d *db.DB, r io.Reader) (int, error) {
	rows, err := parseAdminCodes(r)
	if err != nil {
		return 0, fmt.Errorf("parse admin2 codes: %w", err)
	}

	err = d.WithTx(ctx, func(tx *db.Queries) error {
		if _, txErr := tx.DeleteAllGeonameAdmin2(ctx); txErr != nil {
			return txErr
		}
		for _, row := range rows {
			if txErr := tx.InsertGeonameAdmin2(ctx, db.InsertGeonameAdmin2Params{
				Code:      row.Code,
				Name:      row.Name,
				Geonameid: row.Geonameid,
			}); txErr != nil {
				return txErr
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("import admin2 codes: %w", err)
	}

	logg.Info(ctx, "Admin2 codes imported", "rows", len(rows))
	return len(rows), nil
}

// parseAdminCodes parses tab-delimited admin code rows from r.
// Format: code<tab>name<tab>asciiname<tab>geonameid
func parseAdminCodes(r io.Reader) ([]adminCodeRow, error) {
	scanner := bufio.NewScanner(r)
	var rows []adminCodeRow

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < adminCodeColumns {
			continue
		}

		geonameid, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			continue
		}

		rows = append(rows, adminCodeRow{
			Code:      fields[0],
			Name:      fields[1],
			Geonameid: geonameid,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return rows, nil
}
