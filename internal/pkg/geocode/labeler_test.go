package geocode

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/track"
)

func TestFormatLabel_single(t *testing.T) {
	label := formatLabel([]place{
		{Name: "Zurich", Admin1Name: "Zurich", CountryCode: "CH"},
	})
	require.Equal(t, "Zurich, Zurich, CH", label)
}

func TestFormatLabel_twoSameCountry(t *testing.T) {
	label := formatLabel([]place{
		{Name: "Zurich", Admin1Name: "Zurich", CountryCode: "CH"},
		{Name: "Bern", Admin1Name: "Bern", CountryCode: "CH"},
	})
	require.Equal(t, "Zurich - Bern, CH", label)
}

func TestFormatLabel_twoSameRegion(t *testing.T) {
	label := formatLabel([]place{
		{Name: "Zurich", Admin1Name: "Zurich", CountryCode: "CH"},
		{Name: "Winterthur", Admin1Name: "Zurich", CountryCode: "CH"},
	})
	require.Equal(t, "Zurich - Winterthur, Zurich, CH", label)
}

func TestFormatLabel_multiDifferentCountry(t *testing.T) {
	label := formatLabel([]place{
		{Name: "Basel", Admin1Name: "Basel-Stadt", CountryCode: "CH"},
		{Name: "Freiburg", Admin1Name: "Baden-Wuerttemberg", CountryCode: "DE"},
	})
	require.Equal(t, "Basel - Freiburg (CH/DE)", label)
}

func TestFormatLabel_empty(t *testing.T) {
	require.Equal(t, "", formatLabel(nil))
}

func TestFormatLabel_threePlaces(t *testing.T) {
	label := formatLabel([]place{
		{Name: "Zurich", Admin1Name: "Zurich", CountryCode: "CH"},
		{Name: "Lucerne", Admin1Name: "Lucerne", CountryCode: "CH"},
		{Name: "Bern", Admin1Name: "Bern", CountryCode: "CH"},
	})
	require.Equal(t, "Zurich - Bern, CH", label)
}

func TestEvenSample(t *testing.T) {
	pts := make(track.Points, 20)
	for i := range pts {
		pts[i] = track.Point{Lat: float64(i), Lon: 0}
	}

	sampled := evenSample(pts, 5)
	require.Len(t, sampled, 5)
	require.InDelta(t, 0, sampled[0].Lat, 0.01)
	require.InDelta(t, 19, sampled[4].Lat, 0.01)
}

func TestEvenSample_fewerThanN(t *testing.T) {
	pts := track.Points{{Lat: 1}, {Lat: 2}}
	sampled := evenSample(pts, 10)
	require.Len(t, sampled, 2)
}

// createTestTrack inserts a minimal track with a GPX blob and returns the track UUID.
func createTestTrack(t *testing.T, ctx context.Context, d *db.DB) string {
	t.Helper()

	gpx := `<?xml version="1.0"?>
<gpx version="1.1" creator="test">
  <trk><name>Test</name><trkseg>
    <trkpt lat="47.37" lon="8.55"><ele>400</ele></trkpt>
    <trkpt lat="46.95" lon="7.45"><ele>540</ele></trkpt>
    <trkpt lat="47.56" lon="7.57"><ele>245</ele></trkpt>
  </trkseg></trk>
</gpx>`

	content := []byte(gpx)

	var trackUUID string
	err := d.WithTx(ctx, func(tx *db.Queries) error {
		b, txErr := blob.Create(ctx, tx, content, blob.CompressionZstd)
		if txErr != nil {
			return txErr
		}

		tr, txErr := tx.CreateTrack(ctx, db.CreateTrackParams{
			Uuid:             "test-track-001",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			UserID:           "test-user",
			BlobID:           b.ID,
			FileFormat:       0,
			OriginalFilename: "test.gpx",
			Name:             "Test Track",
			Sport:            0,
			SubSport:         0,
			TotalDistanceM:   100000,
			TotalAscentM:     500,
		})
		if txErr != nil {
			return txErr
		}
		trackUUID = tr.Uuid
		return nil
	})
	require.NoError(t, err)
	return trackUUID
}

func TestLabelerRun(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	// Import some geoname data.
	geoData := strings.Join([]string{
		"2657896\tZurich\tZurich\t\t47.36667\t8.55\tP\tPPLA\tCH\t\tZH\t112\t261\t\t415367\t408\t410\tEurope/Zurich\t2024-09-08",
		"2660646\tBern\tBern\t\t46.94809\t7.44744\tP\tPPLC\tCH\t\tBE\t246\t2546\t\t133883\t540\t542\tEurope/Zurich\t2024-09-08",
		"2661552\tBasel\tBasel\t\t47.55839\t7.57327\tP\tPPLA\tCH\t\tBS\t1200\t2701\t\t177654\t245\t279\tEurope/Zurich\t2024-09-08",
	}, "\n")
	n, err := importFromReader(ctx, d, strings.NewReader(geoData))
	require.NoError(t, err)
	require.Equal(t, 3, n)

	// Import admin1 codes.
	admin1Data := strings.Join([]string{
		"CH.ZH\tZurich\tZurich\t2657895",
		"CH.BE\tBern\tBern\t2661551",
		"CH.BS\tBasel-Stadt\tBasel-Stadt\t2661602",
	}, "\n")
	_, err = ImportAdmin1Codes(ctx, d, strings.NewReader(admin1Data))
	require.NoError(t, err)

	// Create a test user (needed for FK).
	err = d.WithTx(ctx, func(tx *db.Queries) error {
		_, txErr := tx.CreateUser(ctx, db.CreateUserParams{
			Uuid:         "test-user",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Email:        "test@example.com",
			Name:         "TestUser",
			PasswordHash: "hash",
		})
		return txErr
	})
	require.NoError(t, err)

	trackUUID := createTestTrack(t, ctx, d)

	labeler := NewLabeler(d)
	err = labeler.Run(ctx, LabelerArgs{TrackID: trackUUID})
	require.NoError(t, err)

	row, err := d.QueryRO().GetTrackGeoname(ctx, trackUUID)
	require.NoError(t, err)
	require.NotEmpty(t, row.Label)
	t.Logf("Generated label: %s", row.Label)
}

func TestLabelerRun_missingTrack(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	labeler := NewLabeler(d)
	err := labeler.Run(ctx, LabelerArgs{TrackID: "nonexistent"})
	require.NoError(t, err)
}

func TestLabelerRun_noGeonameData(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	// Create user and track but no geoname data.
	err := d.WithTx(ctx, func(tx *db.Queries) error {
		_, txErr := tx.CreateUser(ctx, db.CreateUserParams{
			Uuid:         "test-user",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Email:        "test@example.com",
			Name:         "TestUser",
			PasswordHash: "hash",
		})
		return txErr
	})
	require.NoError(t, err)

	trackUUID := createTestTrack(t, ctx, d)

	labeler := NewLabeler(d)
	err = labeler.Run(ctx, LabelerArgs{TrackID: trackUUID})
	require.NoError(t, err)
}
