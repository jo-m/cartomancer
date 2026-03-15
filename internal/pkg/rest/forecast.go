package rest

import (
	"bytes"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/forecast"
	"jo-m.ch/go/detour/internal/pkg/load"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/session"
	"jo-m.ch/go/detour/internal/pkg/track"
)

type forecastPointResponse struct {
	Index             int      `json:"index"`
	DistanceM         float64  `json:"distanceM"`
	Lat               float64  `json:"lat"`
	Lon               float64  `json:"lon"`
	Time              string   `json:"time"`
	TemperatureC      *float64 `json:"temperatureC"`
	PrecipitationRate *float64 `json:"precipitationRate"`
}

type forecastResponse struct {
	ForecastStatus string                  `json:"forecastStatus"`
	Points         []forecastPointResponse `json:"points"`
}

// handleGetTrackForecast returns a weather forecast time series along a track.
// Query params: startTime (RFC3339), speedKmh (float64, average speed in km/h).
func (sv *server) handleGetTrackForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)
	trackUUID := chi.URLParam(r, "uuid")

	t, err := sv.d.QueryRO().GetTrackByUUID(ctx, trackUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "track not found")
			return
		}
		logg.Error(ctx, "failed to get track", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if t.Public == 0 && (user == nil || user.Uuid != t.UserID) {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}

	q := r.URL.Query()

	startTimeStr := q.Get("startTime")
	if startTimeStr == "" {
		writeError(w, http.StatusBadRequest, "startTime is required")
		return
	}
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "startTime must be RFC3339")
		return
	}

	speedStr := q.Get("speedKmh")
	if speedStr == "" {
		writeError(w, http.StatusBadRequest, "speedKmh is required")
		return
	}
	speedKmh, err := strconv.ParseFloat(speedStr, 64)
	if err != nil || speedKmh <= 0 || speedKmh > 200 {
		writeError(w, http.StatusBadRequest, "speedKmh must be a positive number up to 200")
		return
	}

	b, err := blob.Get(ctx, sv.d.QueryRO(), t.BlobID)
	if err != nil {
		logg.Error(ctx, "failed to get track blob", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	src, err := load.Blob(t.OriginalFilename, bytes.NewReader(b.Content))
	if err != nil {
		logg.Error(ctx, "failed to parse track blob", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	tr := track.New(src, 0)
	pts := tr.Points().Subsample(TrackSubsampleM)
	if len(pts) < 2 {
		writeError(w, http.StatusUnprocessableEntity, "track has too few points")
		return
	}

	// Compute cumulative distances for the subsampled points.
	distances := make([]float64, len(pts))
	for i := 1; i < len(pts); i++ {
		distances[i] = distances[i-1] + pts[i-1].MetersTo(&pts[i])
	}

	speedMs := speedKmh / 3.6
	totalDist := distances[len(distances)-1]
	endTime := startTime.Add(time.Duration(totalDist/speedMs) * time.Second)

	bbox := forecast.BBox{
		MinLat: t.BoundsMinLat.Float64,
		MaxLat: t.BoundsMaxLat.Float64,
		MinLon: t.BoundsMinLon.Float64,
		MaxLon: t.BoundsMaxLon.Float64,
	}
	if !t.BoundsMinLat.Valid || !t.BoundsMaxLat.Valid || !t.BoundsMinLon.Valid || !t.BoundsMaxLon.Valid {
		meta := tr.EnhancedMetadata()
		if meta.BoundsMinLat != nil {
			bbox.MinLat = *meta.BoundsMinLat
			bbox.MaxLat = *meta.BoundsMaxLat
			bbox.MinLon = *meta.BoundsMinLon
			bbox.MaxLon = *meta.BoundsMaxLon
		}
	}

	h, err := forecast.Load(ctx, sv.d, startTime, endTime, bbox)
	status := "full"
	switch {
	case errors.Is(err, forecast.ErrNoData):
		status = "none"
	case errors.Is(err, forecast.ErrIncomplete):
		status = "partial"
	case err != nil:
		logg.Error(ctx, "failed to load forecast", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	result := make([]forecastPointResponse, len(pts))
	for i, p := range pts {
		pointTime := startTime.Add(time.Duration(distances[i]/speedMs) * time.Second)
		rp := forecastPointResponse{
			Index:     i,
			DistanceM: distances[i],
			Lat:       p.Lat,
			Lon:       p.Lon,
			Time:      pointTime.Format(time.RFC3339),
		}

		if status != "none" {
			tempK := h.Sample("T_2M", pointTime, p.Lat, p.Lon)
			if !math.IsNaN(float64(tempK)) {
				v := float64(tempK) - 273.15
				rp.TemperatureC = &v
			}
			precip := h.Sample("TOT_PR", pointTime, p.Lat, p.Lon)
			if !math.IsNaN(float64(precip)) {
				v := float64(precip)
				rp.PrecipitationRate = &v
			}
		}

		result[i] = rp
	}

	writeJSON(w, http.StatusOK, forecastResponse{ForecastStatus: status, Points: result})
}
