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
	"jo-m.ch/go/detour/internal/pkg/meteo/vars"
	"jo-m.ch/go/detour/internal/pkg/session"
	"jo-m.ch/go/detour/internal/pkg/track"
)

type forecastPointResponse struct {
	Index                    int      `json:"index"`
	DistanceM                float64  `json:"distanceM"`
	Lat                      float64  `json:"lat"`
	Lon                      float64  `json:"lon"`
	Time                     string   `json:"time"`
	TemperatureC             *float64 `json:"temperatureC"`
	PrecipitationRate        *float64 `json:"precipitationRate"`
	WindSpeedMs              *float64 `json:"windSpeedMs"`
	WindDirectionDeg         *float64 `json:"windDirectionDeg"`
	RelativeWindDirectionDeg *float64 `json:"relativeWindDirectionDeg"`
}

// forecastUnits describes the display units for each time series field.
type forecastUnits struct {
	TemperatureC             string `json:"temperatureC"`
	PrecipitationRate        string `json:"precipitationRate"`
	WindSpeedMs              string `json:"windSpeedMs"`
	WindDirectionDeg         string `json:"windDirectionDeg"`
	RelativeWindDirectionDeg string `json:"relativeWindDirectionDeg"`
}

type forecastResponse struct {
	ForecastStatus  string                  `json:"forecastStatus"`
	Attribution     string                  `json:"attribution"`
	AttributionHref string                  `json:"attributionHref"`
	Units           forecastUnits           `json:"units"`
	Points          []forecastPointResponse `json:"points"`
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

	// Compute cumulative distances and travel bearings for the subsampled points.
	distances := make([]float64, len(pts))
	bearings := make([]float64, len(pts))
	for i := 1; i < len(pts); i++ {
		distances[i] = distances[i-1] + pts[i-1].MetersTo(&pts[i])
		bearings[i] = forwardBearing(pts[i-1].Lat, pts[i-1].Lon, pts[i].Lat, pts[i].Lon)
	}
	bearings[0] = bearings[1]

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
			tempK := h.Sample(vars.VarT2m.Name, pointTime, p.Lat, p.Lon)
			if !math.IsNaN(float64(tempK)) {
				v := float64(tempK) - 273.15
				rp.TemperatureC = &v
			}
			precip := h.Sample(vars.VarTotPr.Name, pointTime, p.Lat, p.Lon)
			if !math.IsNaN(float64(precip)) {
				// Convert from kg m-2 s-1 to mm/h (1 kg/m2 = 1 mm, so multiply by 3600).
				v := float64(precip) * 3600
				rp.PrecipitationRate = &v
			}
			uWind := h.Sample(vars.VarU10m.Name, pointTime, p.Lat, p.Lon)
			vWind := h.Sample(vars.VarV10m.Name, pointTime, p.Lat, p.Lon)
			if !math.IsNaN(float64(uWind)) && !math.IsNaN(float64(vWind)) {
				speed := math.Hypot(float64(uWind), float64(vWind))
				rp.WindSpeedMs = &speed
				// Meteorological wind direction: direction wind blows FROM.
				dir := math.Atan2(float64(uWind), float64(vWind))*180/math.Pi + 180
				if dir >= 360 {
					dir -= 360
				}
				rp.WindDirectionDeg = &dir

				// Relative wind direction: 0 = headwind, 180 = tailwind.
				rel := dir - bearings[i]
				rel = math.Mod(rel+360, 360)
				rp.RelativeWindDirectionDeg = &rel
			}
		}

		result[i] = rp
	}

	resp := forecastResponse{
		ForecastStatus: status,
		Units: forecastUnits{
			TemperatureC:             "\u00b0C",
			PrecipitationRate:        "mm/h",
			WindSpeedMs:              "m/s",
			WindDirectionDeg:         "deg",
			RelativeWindDirectionDeg: "deg",
		},
		Points: result,
	}
	if h != nil {
		resp.Attribution = h.Attribution
		resp.AttributionHref = h.AttributionHref
	}
	writeJSON(w, http.StatusOK, resp)
}

// forwardBearing computes the initial bearing in degrees [0, 360) from point 1 to point 2.
func forwardBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1R := lat1 * math.Pi / 180
	lat2R := lat2 * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	y := math.Sin(dLon) * math.Cos(lat2R)
	x := math.Cos(lat1R)*math.Sin(lat2R) - math.Sin(lat1R)*math.Cos(lat2R)*math.Cos(dLon)
	brng := math.Atan2(y, x) * 180 / math.Pi
	return math.Mod(brng+360, 360)
}
