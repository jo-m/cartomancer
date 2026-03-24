package geocode

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/geocode/cols"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/track"
)

func TestTrackBBox(t *testing.T) {
	pts := track.Points{
		{Lat: 46.0, Lon: 7.0},
		{Lat: 48.0, Lon: 9.0},
		{Lat: 47.0, Lon: 8.0},
	}
	bbox := trackBBox(pts, 0.05)
	require.InDelta(t, 45.95, bbox[0], 0.001)
	require.InDelta(t, 48.05, bbox[1], 0.001)
	require.InDelta(t, 6.95, bbox[2], 0.001)
	require.InDelta(t, 9.05, bbox[3], 0.001)
}

func TestPointToSegmentDist_perpendicular(t *testing.T) {
	a := track.Point{Lat: 47.0, Lon: 8.0}
	b := track.Point{Lat: 47.0, Lon: 9.0}
	p := track.Point{Lat: 47.01, Lon: 8.5}

	dist, param := pointToSegmentDist(p, a, b)
	require.InDelta(t, 0.5, param, 0.01)
	require.InDelta(t, 1111, dist, 200) // ~1.1 km for 0.01 degrees latitude.
}

func TestPointToSegmentDist_atEndpointA(t *testing.T) {
	a := track.Point{Lat: 47.0, Lon: 8.0}
	b := track.Point{Lat: 47.0, Lon: 9.0}
	p := track.Point{Lat: 47.01, Lon: 7.5}

	_, param := pointToSegmentDist(p, a, b)
	require.InDelta(t, 0.0, param, 0.001)
}

func TestPointToSegmentDist_atEndpointB(t *testing.T) {
	a := track.Point{Lat: 47.0, Lon: 8.0}
	b := track.Point{Lat: 47.0, Lon: 9.0}
	p := track.Point{Lat: 47.01, Lon: 9.5}

	_, param := pointToSegmentDist(p, a, b)
	require.InDelta(t, 1.0, param, 0.001)
}

func TestPointToSegmentDist_zeroLengthSegment(t *testing.T) {
	a := track.Point{Lat: 47.0, Lon: 8.0}
	p := track.Point{Lat: 47.01, Lon: 8.0}

	dist, param := pointToSegmentDist(p, a, a)
	require.InDelta(t, 0.0, param, 0.001)
	require.Greater(t, dist, 0.0)
}

func TestMinDistToPolyline(t *testing.T) {
	polyline := track.Points{
		{Lat: 47.0, Lon: 7.0},
		{Lat: 47.0, Lon: 8.0},
		{Lat: 47.0, Lon: 9.0},
	}
	p := track.Point{Lat: 47.01, Lon: 8.0}

	dist, frac := minDistToPolyline(p, polyline)
	require.InDelta(t, 0.5, frac, 0.05)
	require.Less(t, dist, 2000.0) // Should be ~1.1 km.
}

func TestMinDistToPolyline_atStart(t *testing.T) {
	polyline := track.Points{
		{Lat: 47.0, Lon: 7.0},
		{Lat: 47.0, Lon: 9.0},
	}
	p := track.Point{Lat: 47.01, Lon: 7.0}

	_, frac := minDistToPolyline(p, polyline)
	require.InDelta(t, 0.0, frac, 0.05)
}

func TestMinDistToPolyline_atEnd(t *testing.T) {
	polyline := track.Points{
		{Lat: 47.0, Lon: 7.0},
		{Lat: 47.0, Lon: 9.0},
	}
	p := track.Point{Lat: 47.01, Lon: 9.0}

	_, frac := minDistToPolyline(p, polyline)
	require.InDelta(t, 1.0, frac, 0.05)
}

func TestLabelBudget(t *testing.T) {
	require.Equal(t, 2, labelBudget(5))
	require.Equal(t, 2, labelBudget(15))
	require.Equal(t, 3, labelBudget(30))
	require.Equal(t, 4, labelBudget(60))
	require.Equal(t, maxWaypoints, labelBudget(500))
}

func TestScoreCandidate_populationMatters(t *testing.T) {
	city := candidate{kind: candidatePlace, population: 100000, trackDist: 500}
	village := candidate{kind: candidatePlace, population: 500, trackDist: 500}

	require.Greater(t, scoreCandidate(&city), scoreCandidate(&village))
}

func TestScoreCandidate_proximityMatters(t *testing.T) {
	close := candidate{kind: candidatePlace, population: 1000, trackDist: 100}
	far := candidate{kind: candidatePlace, population: 1000, trackDist: 2500}

	require.Greater(t, scoreCandidate(&close), scoreCandidate(&far))
}

func TestScoreCandidate_passBonus(t *testing.T) {
	pass := candidate{kind: candidateLandmark, featureCode: cols.FeatureCodePASS, trackDist: 500}
	village := candidate{kind: candidatePlace, population: 500, trackDist: 500}

	require.Greater(t, scoreCandidate(&pass), scoreCandidate(&village))
}

func TestScoreCandidate_peakBonus(t *testing.T) {
	peak := candidate{kind: candidateLandmark, featureCode: cols.FeatureCodePK, trackDist: 500}
	plain := candidate{kind: candidateLandmark, featureCode: "", trackDist: 500}

	require.Greater(t, scoreCandidate(&peak), scoreCandidate(&plain))
}

func TestSuppressionRadius(t *testing.T) {
	landmark := candidate{kind: candidateLandmark}
	village := candidate{kind: candidatePlace, population: 500}
	town := candidate{kind: candidatePlace, population: 50000}
	city := candidate{kind: candidatePlace, population: 500000}

	require.Less(t, suppressionRadius(&landmark), suppressionRadius(&village))
	require.Less(t, suppressionRadius(&village), suppressionRadius(&town))
	require.Less(t, suppressionRadius(&town), suppressionRadius(&city))
}

func TestSelectCandidates_respectsBudget(t *testing.T) {
	var candidates []candidate
	for i := range 20 {
		candidates = append(candidates, candidate{
			name:       fmt.Sprintf("Place%d", i),
			trackFrac:  float64(i) / 19.0,
			trackDist:  500,
			kind:       candidatePlace,
			population: int64(1000 * (20 - i)),
		})
	}
	for i := range candidates {
		candidates[i].score = scoreCandidate(&candidates[i])
	}

	selected := selectCandidates(candidates, 4)
	require.LessOrEqual(t, len(selected), 4)
	require.Greater(t, len(selected), 0)
}

func TestSelectCandidates_orderedByTrackFrac(t *testing.T) {
	candidates := []candidate{
		{name: "End", trackFrac: 1.0, trackDist: 100, kind: candidatePlace, population: 50000},
		{name: "Start", trackFrac: 0.0, trackDist: 100, kind: candidatePlace, population: 50000},
		{name: "Mid", trackFrac: 0.5, trackDist: 100, kind: candidatePlace, population: 50000},
	}
	for i := range candidates {
		candidates[i].score = scoreCandidate(&candidates[i])
	}

	selected := selectCandidates(candidates, 5)
	for i := 1; i < len(selected); i++ {
		require.LessOrEqual(t, selected[i-1].trackFrac, selected[i].trackFrac)
	}
}

func TestSelectCandidates_startAndEndGuaranteed(t *testing.T) {
	candidates := []candidate{
		{name: "Start", trackFrac: 0.05, trackDist: 100, kind: candidatePlace, population: 100},
		{name: "Middle", trackFrac: 0.5, trackDist: 100, kind: candidatePlace, population: 999999},
		{name: "End", trackFrac: 0.95, trackDist: 100, kind: candidatePlace, population: 100},
	}
	for i := range candidates {
		candidates[i].score = scoreCandidate(&candidates[i])
	}

	selected := selectCandidates(candidates, 3)
	names := make([]string, len(selected))
	for i, c := range selected {
		names[i] = c.name
	}
	require.Contains(t, names, "Start")
	require.Contains(t, names, "End")
}

func TestSelectCandidates_suppressesNearby(t *testing.T) {
	candidates := []candidate{
		{name: "BigCity", trackFrac: 0.5, trackDist: 100, kind: candidatePlace, population: 500000},
		{name: "Suburb", trackFrac: 0.52, trackDist: 100, kind: candidatePlace, population: 5000},
	}
	for i := range candidates {
		candidates[i].score = scoreCandidate(&candidates[i])
	}

	selected := selectCandidates(candidates, 5)
	names := make([]string, len(selected))
	for i, c := range selected {
		names[i] = c.name
	}
	require.Contains(t, names, "BigCity")
	require.NotContains(t, names, "Suburb")
}

func TestSelectCandidates_landmarkNotSuppressed(t *testing.T) {
	candidates := []candidate{
		{name: "Town", trackFrac: 0.5, trackDist: 100, kind: candidatePlace, population: 5000},
		{name: "Pass", trackFrac: 0.53, trackDist: 200, kind: candidateLandmark, featureCode: cols.FeatureCodePASS},
	}
	for i := range candidates {
		candidates[i].score = scoreCandidate(&candidates[i])
	}

	selected := selectCandidates(candidates, 5)
	require.Len(t, selected, 2)
}

func TestFormatLabel_single(t *testing.T) {
	label := formatLabel([]candidate{
		{name: "Zurich", admin1Name: "Zurich", countryCode: "CH"},
	})
	require.Equal(t, "Zurich, Zurich, CH", label)
}

func TestFormatLabel_twoSameCountry(t *testing.T) {
	label := formatLabel([]candidate{
		{name: "Zurich", admin1Name: "Zurich", countryCode: "CH"},
		{name: "Bern", admin1Name: "Bern", countryCode: "CH"},
	})
	require.Equal(t, "Zurich - Bern, CH", label)
}

func TestFormatLabel_twoSameRegion(t *testing.T) {
	label := formatLabel([]candidate{
		{name: "Zurich", admin1Name: "Zurich", countryCode: "CH"},
		{name: "Winterthur", admin1Name: "Zurich", countryCode: "CH"},
	})
	require.Equal(t, "Zurich - Winterthur, Zurich, CH", label)
}

func TestFormatLabel_multiDifferentCountry(t *testing.T) {
	label := formatLabel([]candidate{
		{name: "Basel", admin1Name: "Basel-Stadt", countryCode: "CH"},
		{name: "Freiburg", admin1Name: "Baden-Wuerttemberg", countryCode: "DE"},
	})
	require.Equal(t, "Basel - Freiburg (CH/DE)", label)
}

func TestFormatLabel_empty(t *testing.T) {
	require.Equal(t, "", formatLabel(nil))
}

func TestFormatLabel_threeWithIntermediate(t *testing.T) {
	label := formatLabel([]candidate{
		{name: "Zurich", countryCode: "CH", trackFrac: 0.0, score: 5},
		{name: "Grimselpass", countryCode: "CH", trackFrac: 0.5, score: 4},
		{name: "Bern", countryCode: "CH", trackFrac: 1.0, score: 5},
	})
	require.Equal(t, "Zurich - Grimselpass - Bern, CH", label)
}

func TestFormatLabel_truncatesIntermediates(t *testing.T) {
	// Build a label that would exceed maxLabelLen with all intermediates.
	selected := []candidate{
		{name: "Alexandroupolis", countryCode: "GR", trackFrac: 0.0, score: 5},
		{name: "Thessaloniki", countryCode: "GR", trackFrac: 0.3, score: 3},
		{name: "Ioannina", countryCode: "GR", trackFrac: 0.5, score: 2},
		{name: "Konstantinoupolis", countryCode: "GR", trackFrac: 0.7, score: 1},
		{name: "Athens", countryCode: "GR", trackFrac: 1.0, score: 5},
	}
	label := formatLabel(selected)
	require.LessOrEqual(t, len(label), maxLabelLen)
	require.Contains(t, label, "Alexandroupolis")
	require.Contains(t, label, "Athens")
}

// createTestTrack inserts a minimal track with a GPX blob and returns the track UUID.
func createTestTrack(t *testing.T, ctx context.Context, d *db.DB, pts []struct{ lat, lon float64 }) string {
	t.Helper()

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0"?>
<gpx version="1.1" creator="test">
  <trk><name>Test</name><trkseg>
`)
	totalDist := 0.0
	for i, pt := range pts {
		fmt.Fprintf(&sb, "    <trkpt lat=\"%.5f\" lon=\"%.5f\"><ele>400</ele></trkpt>\n", pt.lat, pt.lon)
		if i > 0 {
			prev := track.Point{Lat: pts[i-1].lat, Lon: pts[i-1].lon}
			cur := track.Point{Lat: pt.lat, Lon: pt.lon}
			totalDist += prev.MetersTo(&cur)
		}
	}
	sb.WriteString("  </trkseg></trk>\n</gpx>")

	content := []byte(sb.String())

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
			TotalDistanceM:   totalDist,
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

func setupTestUser(t *testing.T, ctx context.Context, d *db.DB) {
	t.Helper()
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
}

func importTestGeodata(t *testing.T, ctx context.Context, d *db.DB) {
	t.Helper()

	geoData := strings.Join([]string{
		"2657896\tZurich\tZurich\t\t47.36667\t8.55\tP\tPPLA\tCH\t\tZH\t112\t261\t\t415367\t408\t410\tEurope/Zurich\t2024-09-08",
		"2660646\tBern\tBern\t\t46.94809\t7.44744\tP\tPPLC\tCH\t\tBE\t246\t2546\t\t133883\t540\t542\tEurope/Zurich\t2024-09-08",
		"2661552\tBasel\tBasel\t\t47.55839\t7.57327\tP\tPPLA\tCH\t\tBS\t1200\t2701\t\t177654\t245\t279\tEurope/Zurich\t2024-09-08",
		"6295495\tGrimselpass\tGrimselpass\t\t46.5718\t8.3384\tT\tPASS\tCH\t\tBE\t301\t\t\t0\t2165\t2165\tEurope/Zurich\t2024-01-01",
		"7285299\tUetliberg\tUetliberg\t\t47.3494\t8.4918\tT\tPK\tCH\t\tZH\t112\t\t\t0\t869\t869\tEurope/Zurich\t2024-01-01",
		// Small suburb near Zurich for suppression testing.
		"9999901\tWiedikon\tWiedikon\t\t47.3640\t8.5200\tP\tPPLX\tCH\t\tZH\t112\t261\t\t3000\t410\t410\tEurope/Zurich\t2024-09-08",
	}, "\n")
	n, err := importFromReader(ctx, d, strings.NewReader(geoData))
	require.NoError(t, err)
	require.Equal(t, 6, n)

	admin1Data := strings.Join([]string{
		"CH.ZH\tZurich\tZurich\t2657895",
		"CH.BE\tBern\tBern\t2661551",
		"CH.BS\tBasel-Stadt\tBasel-Stadt\t2661602",
	}, "\n")
	_, err = ImportAdmin1Codes(ctx, d, strings.NewReader(admin1Data))
	require.NoError(t, err)
}

func TestLabelerRun(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	importTestGeodata(t, ctx, d)
	setupTestUser(t, ctx, d)

	trackUUID := createTestTrack(t, ctx, d, []struct{ lat, lon float64 }{
		{47.37, 8.55}, // Near Zurich.
		{46.95, 7.45}, // Near Bern.
		{47.56, 7.57}, // Near Basel.
	})

	labeler := NewLabeler(d)
	err := labeler.Run(ctx, LabelerArgs{TrackID: trackUUID})
	require.NoError(t, err)

	row, err := d.QueryRO().GetTrackGeoname(ctx, trackUUID)
	require.NoError(t, err)
	require.NotEmpty(t, row.Label)
	t.Logf("Generated label: %s", row.Label)

	require.Contains(t, row.Label, "CH")
}

func TestLabelerRun_withLandmark(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	importTestGeodata(t, ctx, d)
	setupTestUser(t, ctx, d)

	// Track goes from Zurich past Grimselpass to Bern.
	trackUUID := createTestTrack(t, ctx, d, []struct{ lat, lon float64 }{
		{47.37, 8.55}, // Near Zurich.
		{46.80, 8.40}, // Approaching pass.
		{46.57, 8.34}, // Near Grimselpass.
		{46.75, 7.90}, // Heading to Bern.
		{46.95, 7.45}, // Near Bern.
	})

	labeler := NewLabeler(d)
	err := labeler.Run(ctx, LabelerArgs{TrackID: trackUUID})
	require.NoError(t, err)

	row, err := d.QueryRO().GetTrackGeoname(ctx, trackUUID)
	require.NoError(t, err)
	t.Logf("Generated label: %s", row.Label)

	require.Contains(t, row.Label, "Grimselpass")
}

func TestLabelerRun_suppressesSuburbs(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	importTestGeodata(t, ctx, d)
	setupTestUser(t, ctx, d)

	// Short track near Zurich that passes close to both Zurich and Wiedikon.
	trackUUID := createTestTrack(t, ctx, d, []struct{ lat, lon float64 }{
		{47.37, 8.55},  // Zurich center.
		{47.365, 8.53}, // Moving through Wiedikon area.
		{47.36, 8.50},  // Still near Zurich.
		{46.95, 7.45},  // Bern.
	})

	labeler := NewLabeler(d)
	err := labeler.Run(ctx, LabelerArgs{TrackID: trackUUID})
	require.NoError(t, err)

	row, err := d.QueryRO().GetTrackGeoname(ctx, trackUUID)
	require.NoError(t, err)
	t.Logf("Generated label: %s", row.Label)

	// Wiedikon (PPLX, pop 3k) is filtered out as a small city subdivision.
	require.Contains(t, row.Label, "Zurich")
	require.NotContains(t, row.Label, "Wiedikon")
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

	setupTestUser(t, ctx, d)
	trackUUID := createTestTrack(t, ctx, d, []struct{ lat, lon float64 }{
		{47.37, 8.55},
		{46.95, 7.45},
		{47.56, 7.57},
	})

	labeler := NewLabeler(d)
	err := labeler.Run(ctx, LabelerArgs{TrackID: trackUUID})
	require.NoError(t, err)
}

func TestIsSuppressed(t *testing.T) {
	bigCity := candidate{trackFrac: 0.5, kind: candidatePlace, population: 500000}
	nearby := candidate{trackFrac: 0.55}

	require.True(t, isSuppressed(nearby, []candidate{bigCity}))

	farAway := candidate{trackFrac: 0.8}
	require.False(t, isSuppressed(farAway, []candidate{bigCity}))
}

func TestSelectCandidates_empty(t *testing.T) {
	require.Nil(t, selectCandidates(nil, 5))
}

func TestSkipFeatureCode(t *testing.T) {
	require.False(t, skipFeatureCode(cols.FeatureCodePPLA, 0))
	require.False(t, skipFeatureCode(cols.FeatureCodePPLC, 50000))

	require.True(t, skipFeatureCode(cols.FeatureCodePPLX, 3000))
	require.True(t, skipFeatureCode(cols.FeatureCodePPLX, 0))
	require.False(t, skipFeatureCode(cols.FeatureCodePPLX, 200000))

	require.True(t, skipFeatureCode(cols.FeatureCodePPLQ, 0))
	require.True(t, skipFeatureCode(cols.FeatureCodePPLW, 0))
}
