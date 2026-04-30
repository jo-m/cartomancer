package forecast

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/db/forecastdb"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo/vars"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

const (
	// summarizerTimeout is the maximum time the summarizer job is allowed to run.
	summarizerTimeout = 30 * time.Minute

	// summarySpeedKmh is the assumed average speed for the journey.
	summarySpeedKmh = 28.0

	// summaryStartOffset is how far in the future the journey is assumed to start.
	summaryStartOffset = 1 * time.Hour

	// summaryBatchSize is how many track UUIDs to fetch per cursor-based batch.
	summaryBatchSize = 100
)

// SummarizerArgs are the arguments for the track forecast summarizer job.
// When TrackUUID is set, only that single track is processed.
// When empty, all tracks needing a forecast update are processed.
type SummarizerArgs struct {
	TrackUUID string `json:"trackUUID,omitempty"`
}

// Kind implements [jobs.Args].
func (SummarizerArgs) Kind() string { return "forecast.summarizer" }

var _ jobs.Args = (*SummarizerArgs)(nil)

// Summarizer computes forecast summaries for all tracks and stores them in the database.
// Use [NewSummarizer] to create an instance.
type Summarizer struct {
	d  *db.DB
	fd *forecastdb.DB
}

// NewSummarizer creates a new [Summarizer] instance.
//
// Parameters:
//   - d: main database for track, blob, and track_forecasts access.
//   - fd: forecast database for weather data queries.
func NewSummarizer(d *db.DB, fd *forecastdb.DB) *Summarizer {
	return &Summarizer{d: d, fd: fd}
}

var _ jobs.Job[SummarizerArgs] = (*Summarizer)(nil)

// Run implements [jobs.Job].
// It looks up the latest forecast reference time, finds all tracks that need
// updating, and computes a weather summary for each one.
// When args.TrackUUID is set, only that single track is processed regardless
// of whether its forecast is already up to date.
func (s *Summarizer) Run(ctx context.Context, args SummarizerArgs) error {
	ctx, cancel := context.WithTimeout(ctx, summarizerTimeout)
	defer cancel()

	refTime, err := s.fd.QueryRO().GetLatestForecastReferenceTime(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		logg.Info(ctx, "no forecast data available, skipping summarizer")
		return nil
	}
	if err != nil {
		return fmt.Errorf("get latest forecast reference time: %w", err)
	}

	startTime := nextFullHour(time.Now()).Add(summaryStartOffset)

	if args.TrackUUID != "" {
		logg.Info(ctx, "computing forecast summary for single track",
			"trackUUID", args.TrackUUID, "referenceTime", refTime, "startTime", startTime)
		return s.summarizeTrack(ctx, args.TrackUUID, refTime, startTime)
	}

	logg.Info(ctx, "computing track forecast summaries", "referenceTime", refTime, "startTime", startTime)

	var succeeded, failed int
	cursor := "" // UUID v7 values are monotonically increasing, so "" sorts before all.
	for {
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled after %d succeeded, %d failed: %w", succeeded, failed, ctx.Err())
		}

		batch, err := s.d.QueryRO().ListTrackUUIDsNeedingForecastBatch(ctx, db.ListTrackUUIDsNeedingForecastBatchParams{
			ForecastReferenceTime: refTime,
			StartTime:             startTime,
			Uuid:                  cursor,
			Limit:                 summaryBatchSize,
		})
		if err != nil {
			return fmt.Errorf("list tracks needing forecast: %w", err)
		}
		if len(batch) == 0 {
			break
		}

		for _, uuid := range batch {
			if ctx.Err() != nil {
				return fmt.Errorf("context cancelled after %d succeeded, %d failed: %w", succeeded, failed, ctx.Err())
			}

			if err := s.summarizeTrack(ctx, uuid, refTime, startTime); err != nil {
				logg.Error(ctx, "failed to summarize track", "uuid", uuid, "err", err)
				failed++
				continue
			}
			succeeded++
		}
		cursor = batch[len(batch)-1]
	}

	if succeeded == 0 && failed == 0 {
		logg.Info(ctx, "all track forecasts up to date", "referenceTime", refTime)
		return nil
	}

	logg.Info(ctx, "track forecast summarizer finished",
		"succeeded", succeeded, "failed", failed)

	if failed > 0 && succeeded == 0 {
		return fmt.Errorf("all %d tracks failed", failed)
	}
	return nil
}

// summarizeTrack computes and stores the forecast summary for a single track.
func (s *Summarizer) summarizeTrack(ctx context.Context, uuid string, refTime, startTime time.Time) error {
	t, err := s.d.QueryRO().GetTrackByUUID(ctx, uuid)
	if err != nil {
		return fmt.Errorf("get track: %w", err)
	}

	pts, err := InterpolatedTrackPoints(t, SummarizerStepM)
	if errors.Is(err, ErrPolylineMissing) {
		logg.Debug(ctx, "track preview polyline not backfilled, skipping", "uuid", uuid)
		return nil
	}
	if err != nil {
		return fmt.Errorf("interpolate track points: %w", err)
	}
	if len(pts) < 2 {
		logg.Debug(ctx, "track has too few points, skipping", "uuid", uuid)
		return nil
	}

	bearings := pts.Bearings()

	speedMs := summarySpeedKmh / 3.6
	totalDist := pts[len(pts)-1].Distance
	endTime := startTime.Add(time.Duration(totalDist/speedMs) * time.Second)

	bbox := trackBBox(t)

	lats := make([]float64, len(pts))
	lons := make([]float64, len(pts))
	for i, p := range pts {
		lats[i] = p.Lat
		lons[i] = p.Lon
	}

	h, err := Load(ctx, s.fd, startTime, endTime, bbox, lats, lons)
	if errors.Is(err, ErrNoData) {
		logg.Debug(ctx, "no forecast data for track", "uuid", uuid)
		return nil
	}
	if err != nil && !errors.Is(err, ErrIncomplete) {
		return fmt.Errorf("load forecast: %w", err)
	}

	summary := computeSummary(h, pts, bearings, startTime, speedMs)

	return s.d.QueryRW().UpsertTrackForecast(ctx, db.UpsertTrackForecastParams{
		TrackUuid:             uuid,
		CreatedAt:             time.Now(),
		ForecastReferenceTime: refTime,
		StartTime:             startTime,
		AvgTemperatureC:       nullFloat(summary.avgTempC),
		TotalPrecipitationMm:  nullFloat(summary.totalPrecipMm),
		WindHeadMs:            nullFloat(summary.windHeadMs),
		WindRightMs:           nullFloat(summary.windRightMs),
		WindTailMs:            nullFloat(summary.windTailMs),
		WindLeftMs:            nullFloat(summary.windLeftMs),
	})
}

// trackSummary holds the computed forecast summary values.
type trackSummary struct {
	avgTempC      float64
	totalPrecipMm float64
	windHeadMs    float64
	windRightMs   float64
	windTailMs    float64
	windLeftMs    float64
}

// computeSummary samples the forecast at each point along the track and
// aggregates temperature, precipitation, and wind into summary values.
// Each point's cumulative distance must be populated on [track.Point.Distance].
func computeSummary(h *Handle, pts track.Points, bearings []float64, startTime time.Time, speedMs float64) trackSummary {
	var (
		tempSum   float64
		tempCount int

		totalPrecipMm float64

		windSum   [4]float64
		windCount [4]int
	)

	for i := range pts {
		pointTime := startTime.Add(time.Duration(pts[i].Distance/speedMs) * time.Second)

		tempK := h.Sample(vars.VarT2m.Name, pointTime, i)
		if !math.IsNaN(float64(tempK)) {
			tempSum += float64(tempK) - 273.15
			tempCount++
		}

		precipRate := h.Sample(vars.VarTotPr.Name, pointTime, i)
		if !math.IsNaN(float64(precipRate)) && i < len(pts)-1 {
			segDurationS := (pts[i+1].Distance - pts[i].Distance) / speedMs
			totalPrecipMm += float64(precipRate) * segDurationS
		}

		uWind := h.Sample(vars.VarU10m.Name, pointTime, i)
		vWind := h.Sample(vars.VarV10m.Name, pointTime, i)
		if !math.IsNaN(float64(uWind)) && !math.IsNaN(float64(vWind)) {
			speed := math.Hypot(float64(uWind), float64(vWind))
			dir := math.Atan2(float64(uWind), float64(vWind))*180/math.Pi + 180
			if dir >= 360 {
				dir -= 360
			}
			rel := math.Mod(dir-bearings[i]+360, 360)
			sector := relativeWindSector(rel)
			windSum[sector] += speed
			windCount[sector]++
		}
	}

	s := trackSummary{
		avgTempC:      math.NaN(),
		totalPrecipMm: totalPrecipMm,
		windHeadMs:    math.NaN(),
		windRightMs:   math.NaN(),
		windTailMs:    math.NaN(),
		windLeftMs:    math.NaN(),
	}

	if tempCount > 0 {
		s.avgTempC = tempSum / float64(tempCount)
	}

	for i, count := range windCount {
		if count > 0 {
			avg := windSum[i] / float64(count)
			switch i {
			case 0:
				s.windHeadMs = avg
			case 1:
				s.windRightMs = avg
			case 2:
				s.windTailMs = avg
			case 3:
				s.windLeftMs = avg
			}
		}
	}

	return s
}

// relativeWindSector maps a relative wind direction in [0, 360) to one of 4 sectors:
// 0 = head [315, 45), 1 = right [45, 135), 2 = tail [135, 225), 3 = left [225, 315).
func relativeWindSector(relDeg float64) int {
	// Shift by 45 so head (centered at 0) starts at 0.
	shifted := math.Mod(relDeg+45, 360)
	return int(shifted / 90)
}

// trackBBox extracts the bounding box from a track database row.
func trackBBox(t db.Track) BBox {
	return BBox{
		MinLat: t.BoundsMinLat.Float64,
		MaxLat: t.BoundsMaxLat.Float64,
		MinLon: t.BoundsMinLon.Float64,
		MaxLon: t.BoundsMaxLon.Float64,
	}
}

// nextFullHour returns the start of the next full hour after t.
// For example, 09:34 becomes 10:00, and 10:00 stays 10:00.
func nextFullHour(t time.Time) time.Time {
	truncated := t.Truncate(time.Hour)
	if truncated.Equal(t) {
		return t
	}
	return truncated.Add(time.Hour)
}

// nullFloat converts a float64 to sql.NullFloat64, treating NaN as invalid/null.
func nullFloat(v float64) sql.NullFloat64 {
	if math.IsNaN(v) {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: v, Valid: true}
}
