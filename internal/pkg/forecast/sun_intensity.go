package forecast

import (
	"math"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/meteo/vars"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// Sun intensity index computation.
//
// The intensity index integrates the broadband downward shortwave components
// ASWDIR_S (direct) and ASWDIFD_S (diffuse downward) along the track in time,
// using the assumed constant speed to convert distance steps into segment
// durations. Both inputs are per-step means in W/m^2: [forecast.Load]
// de-averages the running-mean form that ICON-CH1-EPS publishes for
// [vars.VarAswdirS] and [vars.VarAswdifdS] before they reach this function.
// Sampled irradiances below [sunIntensityThresholdWm2] contribute zero to
// the integral; the resulting dose is multiplied by [sunIntensityScale] and
// clamped to [[sunIntensityMin], [sunIntensityMax]] so the output is a
// dimensionless 1..10 index suitable for display.

// TODO: calibrate sunIntensityThresholdWm2 and sunIntensityScale against
// real-world rides; the current values are placeholders. The threshold is
// meant to approximate the irradiance below which solar load does not
// meaningfully tax the rider; the scale is meant to map a typical sunny
// multi-hour ride dose to ~10 and a fully shaded/overcast ride to ~1.
const (
	// sunIntensityThresholdWm2 is the irradiance below which a sampled point
	// contributes zero to the integrated sun dose. In W/m^2.
	sunIntensityThresholdWm2 = 150.0

	// sunIntensityScale converts the integrated dose (J/m^2) into the
	// dimensionless intensity index. Tuned so a clear-sky multi-hour ride
	// approaches [sunIntensityMax].
	sunIntensityScale = 1.0 / 1.5e6

	// sunIntensityMin is the lower bound of the displayed index. The integral
	// never goes negative, but clamping to >= 1 keeps the visualization
	// well-defined even on fully overcast rides.
	sunIntensityMin = 1.0

	// sunIntensityMax is the upper bound of the displayed index.
	sunIntensityMax = 10.0
)

// ComputeSunIntensityIndex integrates downward shortwave irradiance along the
// track in time and returns a dimensionless intensity index in
// [[sunIntensityMin], [sunIntensityMax]]. Returns NaN if no point yields a
// valid irradiance sample (e.g. when no forecast data is available).
//
// Parameters:
//   - h: forecast handle with samples for [vars.VarAswdirS] and [vars.VarAswdifdS].
//   - pts: interpolated track points; the cumulative distance must be populated
//     on [track.Point.Distance].
//   - startTime: assumed start time at the first point.
//   - speedMs: assumed constant speed in m/s; must be positive.
func ComputeSunIntensityIndex(h *Handle, pts track.Points, startTime time.Time, speedMs float64) float64 {
	if h == nil || len(pts) < 2 || speedMs <= 0 {
		return math.NaN()
	}

	var (
		dose          float64
		validSamples  int
		segmentsTaken int
	)

	for i := 0; i < len(pts)-1; i++ {
		pointTime := startTime.Add(time.Duration(pts[i].Distance/speedMs) * time.Second)
		segDurationS := (pts[i+1].Distance - pts[i].Distance) / speedMs
		if segDurationS <= 0 {
			continue
		}

		directSW := h.Sample(vars.VarAswdirS.Name, pointTime, i)
		diffuseSW := h.Sample(vars.VarAswdifdS.Name, pointTime, i)
		if math.IsNaN(float64(directSW)) || math.IsNaN(float64(diffuseSW)) {
			continue
		}

		sw := float64(directSW) + float64(diffuseSW)
		validSamples++
		segmentsTaken++

		if sw < sunIntensityThresholdWm2 {
			continue
		}
		dose += sw * segDurationS
	}

	if validSamples == 0 || segmentsTaken == 0 {
		return math.NaN()
	}

	index := dose * sunIntensityScale
	if index < sunIntensityMin {
		index = sunIntensityMin
	}
	if index > sunIntensityMax {
		index = sunIntensityMax
	}
	return index
}
