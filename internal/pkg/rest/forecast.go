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

const forecastIntervalM = 300.0

type forecastPointResponse struct {
	DistanceM         float64 `json:"distanceM"`
	Lat               float64 `json:"lat"`
	Lon               float64 `json:"lon"`
	Time              string  `json:"time"`
	TemperatureC      float64 `json:"temperatureC"`
	PrecipitationRate float64 `json:"precipitationRate"`
}

type forecastResponse struct {
	Points []forecastPointResponse `json:"points"`
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
	pts := tr.Points()
	if len(pts) < 2 {
		writeError(w, http.StatusUnprocessableEntity, "track has too few points")
		return
	}

	interpolated := pts.InterpolateByDistance(forecastIntervalM)
	if len(interpolated) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "track has too few points")
		return
	}

	speedMs := speedKmh / 3.6
	lastPt := interpolated[len(interpolated)-1]
	endTime := startTime.Add(time.Duration(lastPt.DistanceM/speedMs) * time.Second)

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
	if errors.Is(err, forecast.ErrNoData) {
		writeError(w, http.StatusNotFound, "no forecast data available")
		return
	}
	if err != nil && !errors.Is(err, forecast.ErrIncomplete) {
		logg.Error(ctx, "failed to load forecast", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	result := make([]forecastPointResponse, len(interpolated))
	for i, ip := range interpolated {
		pointTime := startTime.Add(time.Duration(ip.DistanceM/speedMs) * time.Second)
		tempK := h.Sample("T_2M", pointTime, ip.Lat, ip.Lon)
		precip := h.Sample("TOT_PR", pointTime, ip.Lat, ip.Lon)

		tempC := float64(tempK) - 273.15
		if math.IsNaN(float64(tempK)) {
			tempC = math.NaN()
		}
		precipRate := float64(precip)
		if math.IsNaN(precipRate) {
			precipRate = 0
		}

		result[i] = forecastPointResponse{
			DistanceM:         ip.DistanceM,
			Lat:               ip.Lat,
			Lon:               ip.Lon,
			Time:              pointTime.Format(time.RFC3339),
			TemperatureC:      tempC,
			PrecipitationRate: precipRate,
		}
	}

	writeJSON(w, http.StatusOK, forecastResponse{Points: result})
}
