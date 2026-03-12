package forecast

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

func TestParseISO8601Duration(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"P0DT00H00M00S", 0, false},
		{"P0DT10H00M00S", 10 * time.Hour, false},
		{"P1DT00H00M00S", 24 * time.Hour, false},
		{"P1DT12H30M00S", 36*time.Hour + 30*time.Minute, false},
		{"P0DT00H00M45S", 45 * time.Second, false},
		{"PT1H", time.Hour, false},
		{"", 0, true},
		{"P1Y", 0, true},
		{"not-a-duration", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseISO8601Duration(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGetLatestForecastReferenceTime_empty(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithDiscardHandler(t.Context())
	_, err := d.QueryRO().GetLatestForecastReferenceTime(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestCreateForecastFile_and_GetLatest(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithDiscardHandler(t.Context())

	blobID, err := uuid.NewV7()
	require.NoError(t, err)
	_, err = blob.Create(ctx, d.QueryRW(), blobID.String(), []byte("grib data"), blob.CompressionZstd)
	require.NoError(t, err)

	refTime := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)
	validTime := refTime.Add(10 * time.Hour)
	const horizonSecs = int64(10 * 3600)

	fileID, err := uuid.NewV7()
	require.NoError(t, err)
	f, err := d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
		Uuid:          fileID.String(),
		CreatedAt:     time.Now(),
		ReferenceTime: refTime,
		ValidTime:     validTime,
		Variable:      "TOT_PREC",
		HorizonSecs:   horizonSecs,
		BoundsMinLat:  sql.NullFloat64{Float64: 45.7, Valid: true},
		BoundsMinLon:  sql.NullFloat64{Float64: 5.9, Valid: true},
		BoundsMaxLat:  sql.NullFloat64{Float64: 47.8, Valid: true},
		BoundsMaxLon:  sql.NullFloat64{Float64: 10.5, Valid: true},
		BlobID:        blobID.String(),
	})
	require.NoError(t, err)
	require.Equal(t, fileID.String(), f.Uuid)
	require.Equal(t, "TOT_PREC", f.Variable)
	require.Equal(t, horizonSecs, f.HorizonSecs)
	require.InDelta(t, 45.7, f.BoundsMinLat.Float64, 1e-9)

	latest, err := d.QueryRO().GetLatestForecastReferenceTime(ctx)
	require.NoError(t, err)
	require.Equal(t, refTime.UTC(), latest.UTC())
}

func TestCreateForecastFile_uniqueConstraint(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithDiscardHandler(t.Context())

	insertBlob := func() string {
		id, err := uuid.NewV7()
		require.NoError(t, err)
		_, err = blob.Create(ctx, d.QueryRW(), id.String(), []byte("grib data"), blob.CompressionNone)
		require.NoError(t, err)
		return id.String()
	}

	refTime := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)

	insert := func(blobID string) error {
		id, err := uuid.NewV7()
		require.NoError(t, err)
		_, err = d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
			Uuid:          id.String(),
			CreatedAt:     time.Now(),
			ReferenceTime: refTime,
			ValidTime:     refTime.Add(time.Hour),
			Variable:      "U_10M",
			HorizonSecs:   3600,
			BlobID:        blobID,
		})
		return err
	}

	require.NoError(t, insert(insertBlob()))
	require.Error(t, insert(insertBlob()), "duplicate (reference_time, variable, horizon_secs) must be rejected")
}
