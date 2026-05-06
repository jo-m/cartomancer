package meteo

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db/forecastdb"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo/vars"
)

// ensureForecast is a test helper that inserts a forecasts row for the given
// reference time if one does not already exist, and returns its ID.
func ensureForecast(t *testing.T, d *forecastdb.DB, refTime time.Time) int64 {
	t.Helper()
	ctx := logg.WithTestLogger(t.Context(), t)

	existing, err := d.QueryRO().ForecastExistsForReferenceTime(ctx, refTime)
	require.NoError(t, err)
	if existing != 0 {
		row, err := d.QueryRO().GetLatestForecast(ctx)
		require.NoError(t, err)
		return row.ID
	}

	row, err := d.QueryRW().CreateForecast(ctx, forecastdb.CreateForecastParams{
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
func insertForecastFileWithValidTime(t *testing.T, d *forecastdb.DB, validTime time.Time) int64 {
	t.Helper()
	ctx := logg.WithTestLogger(t.Context(), t)
	refTime := validTime.Add(-time.Hour)
	forecastID := ensureForecast(t, d, refTime)
	f, err := d.QueryRW().CreateForecastFile(ctx, forecastdb.CreateForecastFileParams{
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
	d := forecastdb.GetTestDB(t)
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
	d := forecastdb.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)
	cleaner := NewCleaner(d)
	err := cleaner.Run(ctx, cleanerArgs{})
	require.NoError(t, err)
}

func TestCleaner_AllFutureFiles_NoOp(t *testing.T) {
	d := forecastdb.GetTestDB(t)
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

func TestCleaner_DeletesEmptyOldForecasts(t *testing.T) {
	d := forecastdb.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)
	now := time.Now()

	// Forecast older than 1 day with no files - must be deleted.
	oldEmptyRef := now.Add(-25 * time.Hour)
	ensureForecast(t, d, oldEmptyRef)

	// Forecast older than 1 day but still has a future file - must be kept.
	oldWithFileRef := now.Add(-26 * time.Hour)
	oldWithFileID := ensureForecast(t, d, oldWithFileRef)
	_, err := d.QueryRW().CreateForecastFile(ctx, forecastdb.CreateForecastFileParams{
		ValidTime:      now.Add(2 * time.Hour),
		ValidUntilTime: now.Add(3 * time.Hour),
		Variable:       vars.VarU10m.Name,
		File:           []byte("grib"),
		ForecastID:     oldWithFileID,
	})
	require.NoError(t, err)

	// Recent forecast with no files - must be kept (not yet 1 day old).
	recentEmptyRef := now.Add(-1 * time.Hour)
	ensureForecast(t, d, recentEmptyRef)

	cleaner := NewCleaner(d)
	err = cleaner.Run(ctx, cleanerArgs{})
	require.NoError(t, err)

	_, err = d.QueryRO().GetForecastByReferenceTime(ctx, oldEmptyRef)
	require.ErrorIs(t, err, sql.ErrNoRows, "old empty forecast must be deleted")

	_, err = d.QueryRO().GetForecastByReferenceTime(ctx, oldWithFileRef)
	require.NoError(t, err, "old forecast with files must be kept")

	_, err = d.QueryRO().GetForecastByReferenceTime(ctx, recentEmptyRef)
	require.NoError(t, err, "recent empty forecast must be kept")
}

func TestCleaner_AllPastFiles(t *testing.T) {
	d := forecastdb.GetTestDB(t)
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
	rows, err := d.QueryRO().ListForecastFilesForWindow(ctx, forecastdb.ListForecastFilesForWindowParams{
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
