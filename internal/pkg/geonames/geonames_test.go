package geonames

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

const subsampleFile = "testdata/TestOnlineSubsampleAllCountries.tsv"

func TestParseLine(t *testing.T) {
	line := "2657896\tZurich\tZurich\tZurich,Zuerich\t47.36667\t8.55\tP\tPPLA\tCH\t\tZH\t112\t261\t\t415367\t408\t410\tEurope/Zurich\t2024-09-08"
	p, err := parseLine(line)
	require.NoError(t, err)
	require.Equal(t, int64(2657896), p.Geonameid)
	require.Equal(t, "Zurich", p.Name)
	require.InDelta(t, 47.36667, p.Latitude, 0.0001)
	require.InDelta(t, 8.55, p.Longitude, 0.01)
	require.Equal(t, "P", p.FeatureClass)
	require.Equal(t, "PPLA", p.FeatureCode)
	require.Equal(t, "CH", p.CountryCode)
}

func TestParseLine_undersea(t *testing.T) {
	line := "123\tSomeRidge\tSomeRidge\t\t10.0\t20.0\tU\tRDGU\tXX\t\t\t\t\t\t0\t\t50\tEurope/London\t2024-01-01"
	_, err := parseLine(line)
	require.ErrorIs(t, err, errSkipped)
}

func TestParseLine_tooFewFields(t *testing.T) {
	_, err := parseLine("123\tPlace\tPlace")
	require.Error(t, err)
}

func TestImportFromReader(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	data := strings.Join([]string{
		"2657896\tZurich\tZurich\t\t47.36667\t8.55\tP\tPPLA\tCH\t\tZH\t112\t261\t\t415367\t408\t410\tEurope/Zurich\t2024-09-08",
		"2660646\tBern\tBern\t\t46.94809\t7.44744\tP\tPPLC\tCH\t\tBE\t246\t2546\t\t133883\t540\t542\tEurope/Zurich\t2024-09-08",
		"2661552\tBasel\tBasel\t\t47.55839\t7.57327\tP\tPPLA\tCH\t\tBS\t1200\t2701\t\t177654\t245\t279\tEurope/Zurich\t2024-09-08",
	}, "\n")

	n, err := importFromReader(ctx, d, strings.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, 3, n)

	count, err := d.QueryRO().CountGeonames(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
}

func TestImportFromReader_replaceExisting(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	line1 := "2657896\tZurich\tZurich\t\t47.36667\t8.55\tP\tPPLA\tCH\t\tZH\t112\t261\t\t415367\t408\t410\tEurope/Zurich\t2024-09-08"
	line2 := "2660646\tBern\tBern\t\t46.94809\t7.44744\tP\tPPLC\tCH\t\tBE\t246\t2546\t\t133883\t540\t542\tEurope/Zurich\t2024-09-08"

	// First import.
	n, err := importFromReader(ctx, d, strings.NewReader(line1+"\n"+line2))
	require.NoError(t, err)
	require.Equal(t, 2, n)

	// Second import replaces data.
	n, err = importFromReader(ctx, d, strings.NewReader(line1))
	require.NoError(t, err)
	require.Equal(t, 1, n)

	count, err := d.QueryRO().CountGeonames(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestImportFromReader_skipsUndersea(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	data := strings.Join([]string{
		"2657896\tZurich\tZurich\t\t47.36667\t8.55\tP\tPPLA\tCH\t\tZH\t112\t261\t\t415367\t408\t410\tEurope/Zurich\t2024-09-08",
		// Undersea feature, should be filtered out during import.
		"101\tSomeRidge\tSomeRidge\t\t47.4\t8.5\tU\tRDGU\tCH\t\t\t\t\t\t0\t\t0\t\t2024-01-01",
	}, "\n")

	n, err := importFromReader(ctx, d, strings.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, 1, n)

	count, err := d.QueryRO().CountGeonames(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestParseSubsampledFile(t *testing.T) {
	data, err := os.ReadFile(subsampleFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Greater(t, len(lines), 100, "subsampled file should have many lines")

	// Every line must either parse successfully or be skipped (e.g. undersea features).
	for i, line := range lines {
		if line == "" {
			continue
		}
		_, err := parseLine(line)
		if errors.Is(err, errSkipped) {
			continue
		}
		require.NoError(t, err, "line %d failed to parse", i)
	}
}

func TestImportSubsampledFile(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	f, err := os.Open(subsampleFile)
	require.NoError(t, err)
	defer f.Close()

	n, err := importFromReader(ctx, d, f)
	require.NoError(t, err)
	require.Greater(t, n, 100, "should import many rows from subsampled file")

	count, err := d.QueryRO().CountGeonames(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(n), count)
}
