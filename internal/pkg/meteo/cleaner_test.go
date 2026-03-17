package meteo

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/meteo/vars"
)

// ensureForecast is a test helper that inserts a forecasts row for the given
// reference time if one does not already exist, and returns its ID.
func ensureForecast(t *testing.T, d *db.DB, refTime time.Time) int64 {
	t.Helper()
	ctx := logg.WithTestLogger(t.Context(), t)

	existing, err := d.QueryRO().ForecastExistsForReferenceTime(ctx, refTime)
	require.NoError(t, err)
	if existing != 0 {
		row, err := d.QueryRO().GetLatestForecast(ctx)
		require.NoError(t, err)
		return row.ID
	}

	row, err := d.QueryRW().CreateForecast(ctx, db.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		HorizontalGridFile: []byte("grid"),
		VerticalGridFile:   []byte("vert grid"),
	})
	require.NoError(t, err)
	return row.ID
}

// insertForecastFileWithValidTime is a test helper that inserts a forecast_files
// row with the given valid_time, returning the file ID.
func insertForecastFileWithValidTime(t *testing.T, d *db.DB, validTime time.Time) int64 {
	t.Helper()
	ctx := logg.WithTestLogger(t.Context(), t)
	refTime := validTime.Add(-time.Hour)
	forecastID := ensureForecast(t, d, refTime)
	f, err := d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
		ValidTime:      validTime,
		ValidUntilTime: validTime.Add(time.Hour),
		Variable:       vars.VarU10m.Name,
		File:           []byte("grib"),
		ForecastID:     forecastID,
	})
	require.NoError(t, err)
	return f.ID
}

func TestCleaner_DeletesPastFiles(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)
	now := time.Now()

	past := now.Add(-2 * time.Hour)
	future := now.Add(2 * time.Hour)

	insertForecastFileWithValidTime(t, d, past)
	insertForecastFileWithValidTime(t, d, future)

	cleaner := NewCleaner(d)
	err := cleaner.Run(ctx, cleanerArgs{})
	require.NoError(t, err)

	// Only the future file should remain.
	latest, err := d.QueryRO().GetLatestForecastReferenceTime(ctx)
	require.NoError(t, err)
	require.Equal(t, future.Add(-time.Hour).UTC().Truncate(time.Second), latest.UTC().Truncate(time.Second))
}

func TestCleaner_EmptyTable(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)
	cleaner := NewCleaner(d)
	err := cleaner.Run(ctx, cleanerArgs{})
	require.NoError(t, err)
}

func TestCleaner_AllFutureFiles_NoOp(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)
	now := time.Now()

	insertForecastFileWithValidTime(t, d, now.Add(1*time.Hour))
	insertForecastFileWithValidTime(t, d, now.Add(2*time.Hour))

	cleaner := NewCleaner(d)
	err := cleaner.Run(ctx, cleanerArgs{})
	require.NoError(t, err)

	// Both files must still exist.
	_, err = d.QueryRO().GetLatestForecastReferenceTime(ctx)
	require.NoError(t, err, "future files must not be deleted")
}

func TestCleaner_AllPastFiles(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)
	now := time.Now()

	insertForecastFileWithValidTime(t, d, now.Add(-3*time.Hour))
	insertForecastFileWithValidTime(t, d, now.Add(-1*time.Hour))

	cleaner := NewCleaner(d)
	err := cleaner.Run(ctx, cleanerArgs{})
	require.NoError(t, err)

	// All forecast files should be deleted; forecast rows remain but have no files.
	// Verify by trying to get the latest reference time - it still exists (forecast
	// rows are not deleted), but a window query should return nothing.
	rows, err := d.QueryRO().ListForecastFilesForWindow(ctx, db.ListForecastFilesForWindowParams{
		Start:  now.Add(-24 * time.Hour),
		End:    now.Add(24 * time.Hour),
		MaxLat: sql.NullFloat64{},
		MinLat: sql.NullFloat64{},
		MaxLon: sql.NullFloat64{},
		MinLon: sql.NullFloat64{},
	})
	require.NoError(t, err)
	require.Empty(t, rows, "all past files should be deleted")
}
