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
// dimensionless 0..12 index suitable for display.
//
// The scale is calibrated against erythemal-UV dose biology. ICON emits
// broadband downward shortwave (~285-2800 nm), of which only a small fraction
// is the erythemally-weighted UV that drives sunburn (McKinlay-Diffey /
// CIE S 007 action spectrum, peaking near 297 nm). The CIE Standard Erythemal
// Dose (SED) is 100 J/m^2 of erythemally-weighted UV; one MED for fair-skinned
// skin type II is roughly 2-3 SED. Index [sunIntensityMax] corresponds to
// [sunIntensityMaxDoseSED] SED of accumulated exposure, well past the
// unprotected-fair-skin sunburn threshold for a multi-hour summer ride.

const (
	// uvErythemalFractionOfSW is the typical fraction of erythemally-weighted
	// UV in broadband downward shortwave at the surface. Real-world values
	// span ~1.5e-4 (high solar zenith angle, winter) to ~3e-4 (summer noon
	// clear sky), and may exceed that under thick clouds since UV is less
	// attenuated than visible light. The midrange value used here integrates
	// reasonably over a multi-hour ride under mixed conditions.
	//
	// Refs: Foyo-Moreno et al. 2003 (Atmos. Res.); WMO/GAW No. 211;
	// ISO 17166:2019 / CIE S 007.
	uvErythemalFractionOfSW = 2.5e-4

	// sunIntensityMaxDoseSED is the erythemal UV dose (in Standard Erythemal
	// Doses, 100 J/m^2 each) at which the index reaches [sunIntensityMax].
	// 25 SED is roughly 10 MED for skin type II, i.e. the severe-burn
	// threshold for unprotected fair skin after a long clear-sky ride.
	sunIntensityMaxDoseSED = 25.0

	// sunIntensityThresholdWm2 is the broadband irradiance below which a
	// sampled point contributes zero to the integrated sun dose.
	sunIntensityThresholdWm2 = 200.0

	// sunIntensityScale converts the integrated broadband dose (J/m^2) into
	// the dimensionless intensity index. Derivation:
	//
	//	1 SED in broadband SW = 100 J/m^2 erythemal / uvErythemalFractionOfSW
	//	                      = 4e5 J/m^2 broadband
	//	dose at sunIntensityMax = sunIntensityMaxDoseSED * (above)
	//	                        = 1e7 J/m^2 broadband
	//	scale                   = sunIntensityMax / 1e7
	//
	// Equivalently, the unclamped index equals dose_SED * sunIntensityMax /
	// sunIntensityMaxDoseSED.
	sunIntensityScale = sunIntensityMax / (sunIntensityMaxDoseSED * (100.0 / uvErythemalFractionOfSW))

	// sunIntensityMin is the lower bound of the displayed index.
	sunIntensityMin = 0.0

	// sunIntensityMax is the upper bound of the displayed index.
	sunIntensityMax = 12.0
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
