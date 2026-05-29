package api

import (
	"errors"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sixdouglas/suncalc"
	"jo-m.ch/go/cartomancer/internal/pkg/forecast"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo/vars"
)

type forecastPointResponse struct {
	DistanceM                float64  `json:"distanceM"`
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

type sunEventResponse struct {
	Type      string  `json:"type"`
	Time      string  `json:"time"`
	DistanceM float64 `json:"distanceM"`
}

// sunIntensityResponse describes the aggregated sun exposure along the track.
type sunIntensityResponse struct {
	// Index is the dimensionless sun intensity index in [0, 1].
	Index float64 `json:"index"`
	// DoseJm2 is the integrated broadband downward shortwave dose along the
	// track in J/m^2.
	DoseJm2 float64 `json:"doseJm2"`
}

type forecastResponse struct {
	ForecastStatus string                  `json:"forecastStatus"`
	Attribution    attributionResponse     `json:"attribution"`
	Units          forecastUnits           `json:"units"`
	Points         []forecastPointResponse `json:"points"`
	SunEvents      []sunEventResponse      `json:"sunEvents"`
	SunIntensity   *sunIntensityResponse   `json:"sunIntensity"`
}

// handleGetTrackForecast returns a weather forecast time series along a track.
// Query params: startTime (RFC3339), speedKmh (float64, average speed in km/h).
func (sv *server) handleGetTrackForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	trackUUID := chi.URLParam(r, "uuid")

	t, ok := sv.getViewableTrack(w, r, trackUUID)
	if !ok {
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

	pts, err := forecast.InterpolatedTrackPoints(t, forecast.LiveStepM)
	if err != nil {
		logg.Error(ctx, "failed to load interpolated track points", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	if len(pts) < 2 {
		writeError(w, http.StatusUnprocessableEntity, "track has too few points")
		return
	}

	// Compute travel bearings for the interpolated points. Cumulative
	// distance is already populated on each point by InterpolatedTrackPoints.
	bearings := pts.Bearings()

	speedMs := speedKmh / 3.6
	totalDist := pts[len(pts)-1].Distance
	endTime := startTime.Add(time.Duration(totalDist/speedMs) * time.Second)

	bbox := forecast.BBox{
		MinLat: t.BoundsMinLat.Float64,
		MaxLat: t.BoundsMaxLat.Float64,
		MinLon: t.BoundsMinLon.Float64,
		MaxLon: t.BoundsMaxLon.Float64,
	}
	if !t.BoundsMinLat.Valid || !t.BoundsMaxLat.Valid || !t.BoundsMinLon.Valid || !t.BoundsMaxLon.Valid {
		// The bounds columns are populated for every track on upload, but old
		// rows may still be NULL. Derive a fallback bbox from the loaded points.
		bbox.MinLat, bbox.MaxLat = pts[0].Lat, pts[0].Lat
		bbox.MinLon, bbox.MaxLon = pts[0].Lon, pts[0].Lon
		for _, p := range pts[1:] {
			bbox.MinLat = math.Min(bbox.MinLat, p.Lat)
			bbox.MaxLat = math.Max(bbox.MaxLat, p.Lat)
			bbox.MinLon = math.Min(bbox.MinLon, p.Lon)
			bbox.MaxLon = math.Max(bbox.MaxLon, p.Lon)
		}
	}

	lats := make([]float64, len(pts))
	lons := make([]float64, len(pts))
	for i, p := range pts {
		lats[i] = p.Lat
		lons[i] = p.Lon
	}

	h, err := forecast.Load(ctx, sv.fd, startTime, endTime, bbox, lats, lons)
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
	for i := range pts {
		pointTime := startTime.Add(time.Duration(pts[i].Distance/speedMs) * time.Second)
		rp := forecastPointResponse{
			DistanceM: pts[i].Distance,
			Time:      pointTime.Format(time.RFC3339),
		}

		if status != "none" {
			tempK := h.Sample(vars.VarT2m.Name, pointTime, i)
			if !math.IsNaN(float64(tempK)) {
				v := float64(tempK) - 273.15
				rp.TemperatureC = &v
			}
			precip := h.Sample(vars.VarTotPr.Name, pointTime, i)
			if !math.IsNaN(float64(precip)) {
				// Convert from kg m-2 s-1 to mm/h (1 kg/m2 = 1 mm, so multiply by 3600).
				v := float64(precip) * 3600
				rp.PrecipitationRate = &v
			}
			uWind := h.Sample(vars.VarU10m.Name, pointTime, i)
			vWind := h.Sample(vars.VarV10m.Name, pointTime, i)
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

	midLat := (bbox.MinLat + bbox.MaxLat) / 2
	midLon := (bbox.MinLon + bbox.MaxLon) / 2
	sunEvents := computeSunEvents(startTime, endTime, midLat, midLon, totalDist)

	var sunIntensity *sunIntensityResponse
	if status != "none" {
		si := forecast.ComputeSunIntensity(h, pts, startTime, speedMs)
		if !math.IsNaN(si.Index) && !math.IsNaN(si.DoseJm2) {
			sunIntensity = &sunIntensityResponse{Index: si.Index, DoseJm2: si.DoseJm2}
		}
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
		Points:       result,
		SunEvents:    sunEvents,
		SunIntensity: sunIntensity,
	}
	if h != nil {
		resp.Attribution = attributionResponse{Text: h.Attribution, Href: h.AttributionHref}
	}
	writeJSON(w, http.StatusOK, resp)
}

// sunEventNames lists the suncalc event types to include in the response.
var sunEventNames = []suncalc.DayTimeName{
	suncalc.Dawn,
	suncalc.Sunrise,
	suncalc.Sunset,
	suncalc.Dusk,
}

// computeSunEvents returns sunrise, sunset, dawn, and dusk events that fall within the ride window.
// Events are interpolated to a distance along the track using a constant speed model.
func computeSunEvents(start, end time.Time, lat, lon, totalDist float64) []sunEventResponse {
	duration := end.Sub(start)

	// Check sun times for each day the ride spans plus padding.
	startDay := start.Add(-24 * time.Hour)
	endDay := end.Add(24 * time.Hour)

	var events []sunEventResponse
	for d := startDay; !d.After(endDay); d = d.Add(24 * time.Hour) {
		times := suncalc.GetTimes(d, lat, lon)
		for _, name := range sunEventNames {
			dt, ok := times[name]
			if !ok || dt.Value.IsZero() {
				continue
			}
			if dt.Value.Before(start) || dt.Value.After(end) {
				continue
			}
			fraction := dt.Value.Sub(start).Seconds() / duration.Seconds()
			distM := fraction * totalDist
			events = append(events, sunEventResponse{
				Type:      string(name),
				Time:      dt.Value.Format(time.RFC3339),
				DistanceM: distM,
			})
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].DistanceM < events[j].DistanceM
	})
	return events
}
