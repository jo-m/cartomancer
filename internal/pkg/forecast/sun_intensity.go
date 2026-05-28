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
// dimensionless 0..1 index suitable for display.
//
// The scale is calibrated against erythemal-UV dose biology. ICON emits
// broadband downward shortwave (~285-2800 nm), of which only a small fraction
// is the erythemally-weighted UV that drives sunburn (McKinlay-Diffey /
// CIE S 007 action spectrum, peaking near 297 nm). The CIE Standard Erythemal
// Dose (SED) is 100 J/m^2 of erythemally-weighted UV; one MED for fair-skinned
// skin type II is roughly 2-3 SED. Index 1.0 ([sunIntensityMax]) corresponds to
// [sunIntensityMaxDoseSED] SED of accumulated exposure, well past the
// unprotected-fair-skin sunburn threshold for a multi-hour summer ride.

const (
	// sunIntensityThresholdWm2 is the broadband irradiance below which a
	// sampled point contributes zero to the integrated sun dose.
	sunIntensityThresholdWm2 = 200.0

	// This approximates a 5h ride at full sun (900W/m^2).
	sunIntensityScale = 1. / (900 * 3600 * 5)

	// sunIntensityMin and sunIntensityMax are the bounds of the output index.
	sunIntensityMin = 0.0
	sunIntensityMax = 1.0
)

// SunIntensity bundles the integrated broadband shortwave dose along a track
// with the dimensionless intensity index derived from it.
type SunIntensity struct {
	// Index is the dimensionless intensity index in [0, 1]. It is clamped,
	// while [DoseJm2] is not.
	Index float64
	// DoseJm2 is the integrated broadband downward shortwave dose along the
	// track in J/m^2. Samples below [sunIntensityThresholdWm2] contribute zero.
	DoseJm2 float64
}

// ComputeSunIntensity integrates downward shortwave irradiance along the track
// in time and returns the broadband dose in J/m^2 together with the
// dimensionless intensity index in [[sunIntensityMin], [sunIntensityMax]].
// Both fields are NaN if no point yields a valid irradiance sample (e.g. when
// no forecast data is available).
//
// Parameters:
//   - h: forecast handle with samples for [vars.VarAswdirS] and [vars.VarAswdifdS].
//   - pts: interpolated track points; the cumulative distance must be populated
//     on [track.Point.Distance].
//   - startTime: assumed start time at the first point.
//   - speedMs: assumed constant speed in m/s; must be positive.
func ComputeSunIntensity(h *Handle, pts track.Points, startTime time.Time, speedMs float64) SunIntensity {
	nanResult := SunIntensity{Index: math.NaN(), DoseJm2: math.NaN()}
	if h == nil || len(pts) < 2 || speedMs <= 0 {
		return nanResult
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
		return nanResult
	}

	index := dose * sunIntensityScale
	if index < sunIntensityMin {
		index = sunIntensityMin
	}
	if index > sunIntensityMax {
		index = sunIntensityMax
	}
	return SunIntensity{Index: index, DoseJm2: dose}
}
