package forecast

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/forecast/vars"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

// insertForecastFileWithValidTime is a test helper that creates a blob and a
// forecast_file row with the given valid_time, returning the file ID.
func insertForecastFileWithValidTime(t *testing.T, d *db.DB, validTime time.Time) int64 {
	t.Helper()
	ctx := logg.WithDiscardHandler(t.Context())
	b, err := blob.Create(ctx, d.QueryRW(), []byte("grib"), blob.CompressionNone)
	require.NoError(t, err)
	refTime := validTime.Add(-time.Hour)
	f, err := d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
		CreatedAt:     time.Now(),
		ReferenceTime: refTime,
		ValidTime:     validTime,
		Variable:      vars.VarU10m.Name,
		BlobID:        b.ID,
	})
	require.NoError(t, err)
	return f.ID
}

func TestCleaner_DeletesPastFiles(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithDiscardHandler(t.Context())
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

	ctx := logg.WithDiscardHandler(t.Context())
	cleaner := NewCleaner(d)
	err := cleaner.Run(ctx, cleanerArgs{})
	require.NoError(t, err)
}

func TestCleaner_AllFutureFiles_NoOp(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithDiscardHandler(t.Context())
	now := time.Now()

	insertForecastFileWithValidTime(t, d, now.Add(1*time.Hour))
	insertForecastFileWithValidTime(t, d, now.Add(2*time.Hour))

	cleaner := NewCleaner(d)
	err := cleaner.Run(ctx, cleanerArgs{})
	require.NoError(t, err)

	// Both files must still exist — check by querying the latest reference_time.
	_, err = d.QueryRO().GetLatestForecastReferenceTime(ctx)
	require.NoError(t, err, "future files must not be deleted")
}

func TestCleaner_AllPastFiles(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithDiscardHandler(t.Context())
	now := time.Now()

	insertForecastFileWithValidTime(t, d, now.Add(-3*time.Hour))
	insertForecastFileWithValidTime(t, d, now.Add(-1*time.Hour))

	cleaner := NewCleaner(d)
	err := cleaner.Run(ctx, cleanerArgs{})
	require.NoError(t, err)

	_, err = d.QueryRO().GetLatestForecastReferenceTime(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows, "all past files should be deleted")
}
