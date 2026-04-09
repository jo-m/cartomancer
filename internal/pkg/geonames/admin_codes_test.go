package geonames

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db/geonamesdb"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

const (
	admin1SubsampleFile = "testdata/TestOnlineSubsampleAdmin1Codes.tsv"
	admin2SubsampleFile = "testdata/TestOnlineSubsampleAdmin2Codes.tsv"
)

func TestParseAdminCodes(t *testing.T) {
	line := "CH.ZH\tZurich\tZurich\t2657895"
	rows, err := parseAdminCodes(strings.NewReader(line))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "CH.ZH", rows[0].Code)
	require.Equal(t, "Zurich", rows[0].Name)
	require.Equal(t, int64(2657895), rows[0].Geonameid)
}

func TestParseAdminCodes_skipMalformed(t *testing.T) {
	data := "CH.ZH\tZurich\tZurich\t2657895\n\nshort\n"
	rows, err := parseAdminCodes(strings.NewReader(data))
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestImportAdmin1Codes(t *testing.T) {
	d := geonamesdb.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	data := strings.Join([]string{
		"CH.ZH\tZurich\tZurich\t2657895",
		"CH.BE\tBern\tBern\t2661551",
		"DE.BY\tBavaria\tBavaria\t2951839",
	}, "\n")

	n, err := ImportAdmin1Codes(ctx, d, strings.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, 3, n)

	count, err := d.QueryRO().CountGeonameAdmin1(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(3), count)

	row, err := d.QueryRO().GetGeonameAdmin1(ctx, "CH.ZH")
	require.NoError(t, err)
	require.Equal(t, "Zurich", row.Name)
}

func TestImportAdmin2Codes(t *testing.T) {
	d := geonamesdb.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	data := strings.Join([]string{
		"CH.ZH.112\tZurich District\tZurich District\t6458798",
		"CH.BE.246\tBern-Mittelland\tBern-Mittelland\t6458783",
	}, "\n")

	n, err := ImportAdmin2Codes(ctx, d, strings.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, 2, n)

	count, err := d.QueryRO().CountGeonameAdmin2(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	row, err := d.QueryRO().GetGeonameAdmin2(ctx, "CH.ZH.112")
	require.NoError(t, err)
	require.Equal(t, "Zurich District", row.Name)
}

func TestImportAdmin1Codes_replaceExisting(t *testing.T) {
	d := geonamesdb.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	data1 := "CH.ZH\tZurich\tZurich\t2657895\nCH.BE\tBern\tBern\t2661551"
	n, err := ImportAdmin1Codes(ctx, d, strings.NewReader(data1))
	require.NoError(t, err)
	require.Equal(t, 2, n)

	data2 := "CH.ZH\tZurich\tZurich\t2657895"
	n, err = ImportAdmin1Codes(ctx, d, strings.NewReader(data2))
	require.NoError(t, err)
	require.Equal(t, 1, n)

	count, err := d.QueryRO().CountGeonameAdmin1(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestImportSubsampledAdmin1(t *testing.T) {
	d := geonamesdb.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	f, err := os.Open(admin1SubsampleFile)
	require.NoError(t, err)
	defer f.Close()

	n, err := ImportAdmin1Codes(ctx, d, f)
	require.NoError(t, err)
	require.Greater(t, n, 20, "should import many admin1 rows from subsampled file")

	count, err := d.QueryRO().CountGeonameAdmin1(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(n), count)
}

func TestImportSubsampledAdmin2(t *testing.T) {
	d := geonamesdb.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	f, err := os.Open(admin2SubsampleFile)
	require.NoError(t, err)
	defer f.Close()

	n, err := ImportAdmin2Codes(ctx, d, f)
	require.NoError(t, err)
	require.Greater(t, n, 100, "should import many admin2 rows from subsampled file")

	count, err := d.QueryRO().CountGeonameAdmin2(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(n), count)
}
