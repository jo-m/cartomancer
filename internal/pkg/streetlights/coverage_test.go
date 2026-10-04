package streetlights

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/require"
	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// coverageTestAttribution is the data source credit used by the coverage
// tests.
var coverageTestAttribution = attribute.Attribution{Author: "Test Office", Source: "https://example.com/lamps"}

// testLamp is a streetlight to insert in the coverage tests.
type testLamp struct {
	ll   orb.Point
	attr attribute.Attribution
}

// insertLamps inserts the given lamps as streetlight rows.
func insertLamps(t *testing.T, d *db.DB, lamps ...testLamp) {
	t.Helper()
	for i, l := range lamps {
		err := d.WithTx(t.Context(), func(tx *db.Queries) error {
			return Insert(t.Context(), tx, StreetlightInsert{
				SourceID:    fmt.Sprintf("lamp-%d", i),
				InsertedBy:  "test",
				Geometry:    geojson.NewGeometry(l.ll),
				Attribution: l.attr,
			}, time.Now())
		})
		require.NoError(t, err)
	}
}

// seedRawStreetlight inserts a streetlight row with an arbitrary geometry
// string, bypassing the validation of [Insert].
func seedRawStreetlight(t *testing.T, d *db.DB, cell int64, geometry string) {
	t.Helper()
	ctx := t.Context()
	err := d.WithTx(ctx, func(q *db.Queries) error {
		attrID, err := q.UpsertStreetlightAttribution(ctx, db.UpsertStreetlightAttributionParams{
			Attribution:     "Raw",
			AttributionHref: "https://raw.example",
		})
		if err != nil {
			return err
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		return q.InsertStreetlight(ctx, db.InsertStreetlightParams{
			Uuid:          id.String(),
			SourceID:      "raw",
			InsertedBy:    "test",
			CreatedAt:     time.Now(),
			AttributionID: attrID,
			Cell:          cell,
			Geometry:      geometry,
		})
	})
	require.NoError(t, err)
}

// round5 rounds a coordinate to the 1e-5 degree precision of the varint
// encoder, so that downstream distances are exactly reproducible.
func round5(v float64) float64 {
	return math.Round(v*1e5) / 1e5
}

// encodeTrack computes cumulative distances for pts and encodes them as a
// dp5m varint polyline.
func encodeTrack(t *testing.T, pts track.Points) db.Track {
	t.Helper()
	for i := 1; i < len(pts); i++ {
		pts[i].Distance = pts[i-1].Distance + pts[i-1].MetersTo(&pts[i])
	}
	blob, err := track.EncodeVarint(pts)
	require.NoError(t, err)
	return db.Track{PolylineDp5mVarint: blob, TotalDistanceM: pts[len(pts)-1].Distance}
}

// straightTrack builds a north-south track of approximately lengthM meters
// with vertices at quarter positions. The exact length depends on the
// great-circle distance between the chosen coordinates; tests derive their
// expectations from the decoded samples rather than from lengthM.
func straightTrack(t *testing.T, lat, lon, lengthM float64) db.Track {
	t.Helper()
	// The conversion factor only needs to be close; the distances are
	// measured below, not assumed.
	const metersPerDegreeLatApprox = 111195.0
	pts := track.Points{{Lat: round5(lat), Lon: round5(lon)}}
	for _, frac := range []float64{0.25, 0.5, 0.75, 1} {
		pts = append(pts, track.Point{Lat: round5(lat + frac*lengthM/metersPerDegreeLatApprox), Lon: round5(lon)})
	}
	return encodeTrack(t, pts)
}

// eastWestTrack builds an east-west track of approximately lengthM meters at
// the given latitude, analogous to [straightTrack].
func eastWestTrack(t *testing.T, lat, lon, lengthM float64) db.Track {
	t.Helper()
	mPerDegLon := 111195.0 * math.Cos(lat*math.Pi/180)
	pts := track.Points{{Lat: round5(lat), Lon: round5(lon)}}
	for _, frac := range []float64{0.25, 0.5, 0.75, 1} {
		pts = append(pts, track.Point{Lat: round5(lat), Lon: round5(lon + frac*lengthM/mPerDegLon)})
	}
	return encodeTrack(t, pts)
}

// samplesOf decodes the track's polyline and returns its interpolated samples.
func samplesOf(t *testing.T, tr db.Track) []track.InterpolatedPoint {
	t.Helper()
	pts, err := track.DecodeVarint(tr.PolylineDp5mVarint)
	require.NoError(t, err)
	samples := pts.InterpolateByDistance(SampleStepM)
	require.NotEmpty(t, samples)
	return samples
}

// sampleAt returns the interpolated sample nearest to the given cumulative
// distance.
func sampleAt(t *testing.T, tr db.Track, distM float64) track.InterpolatedPoint {
	t.Helper()
	samples := samplesOf(t, tr)
	best := samples[0]
	for _, s := range samples {
		if math.Abs(s.DistanceM-distM) < math.Abs(best.DistanceM-distM) {
			best = s
		}
	}
	return best
}

// midPoint returns the coordinate halfway between two samples.
func midPoint(a, b track.InterpolatedPoint) orb.Point {
	return orb.Point{(a.Lon + b.Lon) / 2, (a.Lat + b.Lat) / 2}
}

// metersPerDegreeLon measures the great-circle length of one degree of
// longitude at the given coordinate.
func metersPerDegreeLon(lat, lon float64) float64 {
	return metersBetween(lat, lon, lat, lon+0.001) / 0.001
}

// offsetEast returns the coordinate reached by moving eastM meters east from
// (lat, lon).
func offsetEast(lat, lon, eastM float64) orb.Point {
	return orb.Point{lon + eastM/metersPerDegreeLon(lat, lon), lat}
}

// coversDistance reports whether any stretch contains the given distance.
func coversDistance(stretches []LitStretch, distM float64) bool {
	for _, s := range stretches {
		if distM >= s.StartDistanceM && distM <= s.EndDistanceM {
			return true
		}
	}
	return false
}

func TestComputeTrackCoverage_EmptyTable(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	tr := straightTrack(t, 47.3, 8.5, 1000)
	res, err := ComputeTrackCoverage(t.Context(), d, tr)
	require.NoError(t, err)

	samples := samplesOf(t, tr)
	span := samples[len(samples)-1].DistanceM - samples[0].DistanceM
	require.InDelta(t, 1000, span, 5)
	require.Equal(t, span, res.TotalDistanceM)
	require.Zero(t, res.LitDistanceM)
	require.NotNil(t, res.Stretches)
	require.Empty(t, res.Stretches)
	require.NotNil(t, res.Attributions)
	require.Empty(t, res.Attributions)
}

func TestComputeTrackCoverage_LitRadiusThreshold(t *testing.T) {
	// A lamp 19.9 m to the side of the sample at 20 m lights it, 20.1 m does
	// not. The next samples are sqrt(19.9^2 + 25) m away, so only the sample
	// at 20 m can be lit.
	for _, tc := range []struct {
		name     string
		lateralM float64
		lit      bool
	}{
		{name: "inside", lateralM: 19.9, lit: true},
		{name: "outside", lateralM: 20.1, lit: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := db.GetTestDB(t)
			t.Cleanup(func() { d.Close() })

			tr := straightTrack(t, 47.3, 8.5, 40)
			s20 := sampleAt(t, tr, 20)
			lampPoint := offsetEast(s20.Lat, s20.Lon, tc.lateralM)

			// Setup sanity: the lamp really is just inside or just outside
			// the radius, so the assertions below test the boundary logic.
			dist := metersBetween(s20.Lat, s20.Lon, lampPoint.Lat(), lampPoint.Lon())
			if tc.lit {
				require.Less(t, dist, LitRadiusM)
			} else {
				require.Greater(t, dist, LitRadiusM)
			}

			insertLamps(t, d, testLamp{ll: lampPoint, attr: coverageTestAttribution})

			res, err := ComputeTrackCoverage(t.Context(), d, tr)
			require.NoError(t, err)
			if !tc.lit {
				require.Zero(t, res.LitDistanceM)
				require.Empty(t, res.Stretches)
				return
			}

			// Only the sample at 20 m is lit, so the stretch spans the
			// midpoints to the samples at 15 m and 25 m.
			s15 := sampleAt(t, tr, 15)
			s25 := sampleAt(t, tr, 25)
			require.Greater(t, metersBetween(s15.Lat, s15.Lon, lampPoint.Lat(), lampPoint.Lon()), LitRadiusM)
			require.Greater(t, metersBetween(s25.Lat, s25.Lon, lampPoint.Lat(), lampPoint.Lon()), LitRadiusM)
			require.Equal(t, []LitStretch{{StartDistanceM: 17.5, EndDistanceM: 22.5}}, res.Stretches)
			require.Equal(t, 5.0, res.LitDistanceM)
		})
	}
}

func TestComputeTrackCoverage_StretchBoundaries(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	tr := straightTrack(t, 47.3, 8.5, 1000)

	// The lamp sits between the samples at 500 m and 505 m, 2.5 m from each,
	// so the lit samples are those between 482.5 m and 522.5 m and the
	// stretch boundaries land exactly on the midpoints to the first unlit
	// samples at 480 m and 525 m.
	lampPoint := midPoint(sampleAt(t, tr, 500), sampleAt(t, tr, 505))
	for distM, lit := range map[float64]bool{485: true, 520: true, 480: false, 525: false} {
		s := sampleAt(t, tr, distM)
		lampDist := metersBetween(s.Lat, s.Lon, lampPoint.Lat(), lampPoint.Lon())
		if lit {
			require.Less(t, lampDist, LitRadiusM, "sample at %v m", distM)
		} else {
			require.Greater(t, lampDist, LitRadiusM, "sample at %v m", distM)
		}
	}

	insertLamps(t, d, testLamp{ll: lampPoint, attr: coverageTestAttribution})

	res, err := ComputeTrackCoverage(t.Context(), d, tr)
	require.NoError(t, err)
	require.Equal(t, []LitStretch{{StartDistanceM: 482.5, EndDistanceM: 522.5}}, res.Stretches)
	require.Equal(t, 40.0, res.LitDistanceM)
	require.InDelta(t, 1000, res.TotalDistanceM, 5)
}

func TestComputeTrackCoverage_MergedLamps(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	tr := straightTrack(t, 47.3, 8.5, 1000)

	// Two lamps 30 m apart light overlapping sample ranges, so their
	// stretches must merge into one.
	lampA := midPoint(sampleAt(t, tr, 500), sampleAt(t, tr, 505))
	lampB := midPoint(sampleAt(t, tr, 530), sampleAt(t, tr, 535))
	for _, tc := range []struct {
		lamp   orb.Point
		within []float64
		beyond []float64
	}{
		{lamp: lampA, within: []float64{485, 520}, beyond: []float64{480, 525}},
		{lamp: lampB, within: []float64{515, 550}, beyond: []float64{510, 555}},
	} {
		for _, distM := range tc.within {
			s := sampleAt(t, tr, distM)
			require.Less(t, metersBetween(s.Lat, s.Lon, tc.lamp.Lat(), tc.lamp.Lon()), LitRadiusM)
		}
		for _, distM := range tc.beyond {
			s := sampleAt(t, tr, distM)
			require.Greater(t, metersBetween(s.Lat, s.Lon, tc.lamp.Lat(), tc.lamp.Lon()), LitRadiusM)
		}
	}

	insertLamps(t, d,
		testLamp{ll: lampA, attr: coverageTestAttribution},
		testLamp{ll: lampB, attr: coverageTestAttribution},
	)

	res, err := ComputeTrackCoverage(t.Context(), d, tr)
	require.NoError(t, err)
	require.Equal(t, []LitStretch{{StartDistanceM: 482.5, EndDistanceM: 552.5}}, res.Stretches)
	require.Equal(t, 70.0, res.LitDistanceM)
}

func TestComputeTrackCoverage_FullyLit(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	// A single lamp in the middle of a 30 m track lights every sample.
	tr := straightTrack(t, 47.3, 8.5, 30)
	s15 := sampleAt(t, tr, 15)
	insertLamps(t, d, testLamp{ll: orb.Point{s15.Lon, s15.Lat}, attr: coverageTestAttribution})

	res, err := ComputeTrackCoverage(t.Context(), d, tr)
	require.NoError(t, err)

	samples := samplesOf(t, tr)
	require.Equal(t, samples[len(samples)-1].DistanceM-samples[0].DistanceM, res.TotalDistanceM)
	require.Equal(t, res.TotalDistanceM, res.LitDistanceM)
	require.Len(t, res.Stretches, 1)
	require.Zero(t, res.Stretches[0].StartDistanceM)
	require.Equal(t, res.TotalDistanceM, res.Stretches[0].EndDistanceM)
	require.Equal(t, []Attribution{{
		Text: coverageTestAttribution.Author,
		Href: coverageTestAttribution.Source,
	}}, res.Attributions)
}

func TestComputeTrackCoverage_Attributions(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	tr := straightTrack(t, 47.3, 8.5, 60)
	alpha := attribute.Attribution{Author: "Alpha", Source: "https://a.example"}
	beta := attribute.Attribution{Author: "Beta", Source: "https://b.example"}
	gamma := attribute.Attribution{Author: "Gamma", Source: "https://g.example"}

	s10 := sampleAt(t, tr, 10)
	s30 := sampleAt(t, tr, 30)
	s40 := sampleAt(t, tr, 40)
	s55 := sampleAt(t, tr, 55)

	insertLamps(t, d,
		testLamp{ll: orb.Point{s10.Lon, s10.Lat}, attr: alpha},
		// A second lamp of the same source must not duplicate the credit.
		testLamp{ll: orb.Point{s55.Lon, s55.Lat}, attr: alpha},
		testLamp{ll: orb.Point{s30.Lon, s30.Lat}, attr: beta},
		// This lamp is loaded (its cell is queried) but sits 30 m off the
		// path, lights nothing, and must not be credited.
		testLamp{ll: offsetEast(s40.Lat, s40.Lon, 30), attr: gamma},
	)

	res, err := ComputeTrackCoverage(t.Context(), d, tr)
	require.NoError(t, err)
	require.Equal(t, []Attribution{
		{Text: "Alpha", Href: "https://a.example"},
		{Text: "Beta", Href: "https://b.example"},
	}, res.Attributions)
}

func TestComputeTrackCoverage_UnusableLampGeometry(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })
	ctx := logg.WithDiscardHandler(t.Context())

	tr := straightTrack(t, 47.3, 8.5, 60)
	s20 := sampleAt(t, tr, 20)
	cell20, err := h3.LatLngToCell(h3.LatLng{Lat: s20.Lat, Lng: s20.Lon}, CellResolution)
	require.NoError(t, err)

	// Rows that [Insert] would reject, planted directly so the coverage
	// computation has to skip them.
	seedRawStreetlight(t, d, int64(cell20), "not json")
	seedRawStreetlight(t, d, int64(cell20), `{"type":"LineString","coordinates":[[8.5,47.3],[8.51,47.31]]}`)

	// A lamp at 70 percent of the way from the sample at 15 m to the sample
	// at 20 m, about 18.5 m along the track, lights the samples from 0 m to
	// 35 m; the sample at 40 m is about 21.5 m away.
	s15 := sampleAt(t, tr, 15)
	lampPoint := orb.Point{s15.Lon, s15.Lat + 0.7*(s20.Lat-s15.Lat)}
	for distM, lit := range map[float64]bool{0: true, 35: true, 40: false} {
		s := sampleAt(t, tr, distM)
		lampDist := metersBetween(s.Lat, s.Lon, lampPoint.Lat(), lampPoint.Lon())
		if lit {
			require.Less(t, lampDist, LitRadiusM, "sample at %v m", distM)
		} else {
			require.Greater(t, lampDist, LitRadiusM, "sample at %v m", distM)
		}
	}
	insertLamps(t, d, testLamp{ll: lampPoint, attr: coverageTestAttribution})

	res, err := ComputeTrackCoverage(ctx, d, tr)
	require.NoError(t, err)
	require.Equal(t, []LitStretch{{StartDistanceM: 0, EndDistanceM: 37.5}}, res.Stretches)
	require.Equal(t, 37.5, res.LitDistanceM)
	// Only the valid lamp's source appears; the raw rows were skipped.
	require.Equal(t, []Attribution{{
		Text: coverageTestAttribution.Author,
		Href: coverageTestAttribution.Source,
	}}, res.Attributions)
}

func TestComputeTrackCoverage_ShortPolyline(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	// No polyline at all.
	res, err := ComputeTrackCoverage(t.Context(), d, db.Track{TotalDistanceM: 1234.5})
	require.NoError(t, err)
	require.Equal(t, 1234.5, res.TotalDistanceM)
	require.Zero(t, res.LitDistanceM)
	require.NotNil(t, res.Stretches)
	require.Empty(t, res.Stretches)
	require.NotNil(t, res.Attributions)
	require.Empty(t, res.Attributions)

	// A single-point polyline.
	blob, err := track.EncodeVarint(track.Points{{Lat: 47.3, Lon: 8.5}})
	require.NoError(t, err)
	res, err = ComputeTrackCoverage(t.Context(), d, db.Track{PolylineDp5mVarint: blob, TotalDistanceM: 999})
	require.NoError(t, err)
	require.Equal(t, 999.0, res.TotalDistanceM)
	require.Zero(t, res.LitDistanceM)
	require.Empty(t, res.Stretches)
}

func TestComputeTrackCoverage_NeighbourCellLamp(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	tr := eastWestTrack(t, 47.3, 8.5, 1000)
	samples := samplesOf(t, tr)
	cells, invalid := sampleCells(samples)
	require.Zero(t, invalid)

	// Find a place where the track crosses an H3 cell boundary and put the
	// lamp 12 m past it, so the lamp sits in the next cell while the sample
	// behind the boundary is still within the lit radius.
	idx := -1
	for i := 0; i+1 < len(samples); i++ {
		if cells[i].IsValid() && cells[i+1].IsValid() && cells[i] != cells[i+1] {
			idx = i
			break
		}
	}
	require.NotEqual(t, -1, idx, "track must cross an h3 cell boundary")

	lampPoint := offsetEast(samples[idx].Lat, samples[idx].Lon, 12)
	lampCell, err := h3.LatLngToCell(h3.LatLng{Lat: lampPoint.Lat(), Lng: lampPoint.Lon()}, CellResolution)
	require.NoError(t, err)
	require.NotEqual(t, cells[idx], lampCell, "lamp must be in a different cell than the sample")

	// Setup sanity: the lamp is within the lit radius of the sample.
	lampDist := metersBetween(samples[idx].Lat, samples[idx].Lon, lampPoint.Lat(), lampPoint.Lon())
	require.LessOrEqual(t, lampDist, LitRadiusM)

	insertLamps(t, d, testLamp{ll: lampPoint, attr: coverageTestAttribution})

	res, err := ComputeTrackCoverage(t.Context(), d, tr)
	require.NoError(t, err)
	require.True(t, coversDistance(res.Stretches, samples[idx].DistanceM),
		"sample at %v m, whose cell holds no lamp, must be lit by the lamp across the cell boundary",
		samples[idx].DistanceM)
}

func TestComputeTrackCoverage_LongTrackChunkedQuery(t *testing.T) {
	d := db.GetTestDB(t)
	t.Cleanup(func() { d.Close() })

	tr := straightTrack(t, 47.3, 8.5, 200_000)
	samples := samplesOf(t, tr)
	cells, invalid := sampleCells(samples)
	require.Zero(t, invalid)
	require.Greater(t, len(expandCells(cells)), 500,
		"the query cell list must span multiple database chunk statements")

	res, err := ComputeTrackCoverage(t.Context(), d, tr)
	require.NoError(t, err)
	require.InDelta(t, 200_000, res.TotalDistanceM, 1000)
	require.Zero(t, res.LitDistanceM)
	require.Empty(t, res.Stretches)
	require.Empty(t, res.Attributions)
}

func TestBuildStretches(t *testing.T) {
	samples := make([]track.InterpolatedPoint, 7)
	for i := range samples {
		samples[i] = track.InterpolatedPoint{DistanceM: float64(i) * SampleStepM}
	}

	tests := []struct {
		name          string
		lit           []bool
		wantStretches []LitStretch
		wantLitDistM  float64
	}{
		{
			name:          "all lit",
			lit:           []bool{true, true, true, true, true, true, true},
			wantStretches: []LitStretch{{StartDistanceM: 0, EndDistanceM: 30}},
			wantLitDistM:  30,
		},
		{
			name:          "none lit",
			lit:           []bool{false, false, false, false, false, false, false},
			wantStretches: []LitStretch{},
			wantLitDistM:  0,
		},
		{
			name: "alternating",
			lit:  []bool{true, false, true, false, true, false, true},
			wantStretches: []LitStretch{
				{StartDistanceM: 0, EndDistanceM: 2.5},
				{StartDistanceM: 7.5, EndDistanceM: 12.5},
				{StartDistanceM: 17.5, EndDistanceM: 22.5},
				{StartDistanceM: 27.5, EndDistanceM: 30},
			},
			wantLitDistM: 15,
		},
		{
			name:          "single first sample",
			lit:           []bool{true, false, false, false, false, false, false},
			wantStretches: []LitStretch{{StartDistanceM: 0, EndDistanceM: 2.5}},
			wantLitDistM:  2.5,
		},
		{
			name:          "single last sample",
			lit:           []bool{false, false, false, false, false, false, true},
			wantStretches: []LitStretch{{StartDistanceM: 27.5, EndDistanceM: 30}},
			wantLitDistM:  2.5,
		},
		{
			name:          "single middle sample",
			lit:           []bool{false, false, true, false, false, false, false},
			wantStretches: []LitStretch{{StartDistanceM: 7.5, EndDistanceM: 12.5}},
			wantLitDistM:  5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stretches, litDistM := buildStretches(samples, tc.lit)
			require.Equal(t, tc.wantStretches, stretches)
			require.Equal(t, tc.wantLitDistM, litDistM)
		})
	}
}

func TestWithinSearchBox(t *testing.T) {
	const lat, lon = 47.3, 8.5

	mPerLat := metersBetween(lat, lon, lat+0.001, lon) / 0.001
	mPerLon := metersPerDegreeLon(lat, lon)

	tests := []struct {
		name             string
		lampLat, lampLon float64
		want             bool
	}{
		{name: "same point", lampLat: lat, lampLon: lon, want: true},
		{name: "24 m north", lampLat: lat + 24/mPerLat, lampLon: lon, want: true},
		{name: "26 m north", lampLat: lat + 26/mPerLat, lampLon: lon, want: false},
		{name: "24 m east", lampLat: lat, lampLon: lon + 24/mPerLon, want: true},
		{name: "26 m east", lampLat: lat, lampLon: lon + 26/mPerLon, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, withinSearchBox(lat, lon, tc.lampLat, tc.lampLon))
		})
	}
}

func TestMetersBetween(t *testing.T) {
	require.Zero(t, metersBetween(47.3, 8.5, 47.3, 8.5))

	// One thousandth of a degree of latitude is about 111 m.
	require.InDelta(t, 111.2, metersBetween(47.3, 8.5, 47.301, 8.5), 1.0)
	// One thousandth of a degree of longitude at this latitude is about 75 m.
	require.InDelta(t, 75.4, metersBetween(47.3, 8.5, 47.3, 8.501), 1.0)
}

func TestSampleCellsAndExpand(t *testing.T) {
	samples := []track.InterpolatedPoint{
		{Lat: 47.3, Lon: 8.5},
		{Lat: 47.3, Lon: 8.5},
		{Lat: math.NaN(), Lon: 8.5},
	}

	cells, invalid := sampleCells(samples)
	require.Equal(t, 1, invalid)
	require.True(t, cells[0].IsValid())
	require.Equal(t, cells[0], cells[1])
	require.False(t, cells[2].IsValid())

	// A normal hexagonal cell expands to itself plus its six neighbours.
	expanded := expandCells(cells)
	require.Len(t, expanded, 7)
	require.Contains(t, expanded, int64(cells[0]))
	for i := 1; i < len(expanded); i++ {
		require.Less(t, expanded[i-1], expanded[i], "cells must be sorted and deduplicated")
	}

	// The zero cell used for failed conversions is skipped.
	require.Empty(t, expandCells([]h3.Cell{0}))
}

func TestLampNeighbourhoods(t *testing.T) {
	rows := []db.StreetlightByCell{
		{Cell: 42, Geometry: `{"type":"Point","coordinates":[8.5,47.3]}`, Attribution: "A", AttributionHref: "https://a.example"},
		{Cell: 42, Geometry: "not json", Attribution: "B", AttributionHref: "https://b.example"},
		{Cell: 42, Geometry: `{"type":"LineString","coordinates":[[8.5,47.3],[8.51,47.31]]}`, Attribution: "C", AttributionHref: "https://c.example"},
	}

	byCell, unusable := lampNeighbourhoods(rows)
	require.Equal(t, 2, unusable)
	require.Len(t, byCell, 1)

	lamps := byCell[h3.Cell(42)]
	require.Len(t, lamps, 1)
	require.InDelta(t, 47.3, lamps[0].lat, 1e-9)
	require.InDelta(t, 8.5, lamps[0].lon, 1e-9)
	require.Equal(t, Attribution{Text: "A", Href: "https://a.example"}, lamps[0].attribution)
}
