package forecast

import "math"

// UV exposure estimation.
//
// ICON-CH1-EPS does not publish ultraviolet irradiance directly. The functions
// in this file approximate erythemally-weighted UV from the model's broadband
// downward shortwave (SW) flux components ASWDIR_S (direct) and ASWDIFD_S
// (diffuse downward). The single-coefficient ratio collapses the solar-zenith,
// total-ozone, and aerosol dependences into one constant calibrated for a
// clear-sky summer noon at mid-latitudes; absolute UV index error is on the
// order of +-30% at other geometries. The output is intended as a relative
// "more vs. less UV today" signal, not a substitute for a measured UV index.

const (
	// erythemalUVFraction is the assumed ratio of erythemally-weighted UV
	// irradiance to total downward shortwave irradiance at the surface.
	// Chosen so that SW ~= 1000 W/m^2 (clear-sky noon) maps to UV index ~10.
	erythemalUVFraction = 2.5e-4

	// uviPerErythemalWm2 is the conversion factor from erythemal UV
	// irradiance (W/m^2) to UV index. 1 UVI is defined as 25 mW/m^2.
	uviPerErythemalWm2 = 40.0

	// sedJoulesPerM2 is the energy in one Standard Erythema Dose:
	// 100 J/m^2 of erythemally-weighted UV.
	sedJoulesPerM2 = 100.0
)

// ErythemalUVFromSW returns the erythemally-weighted UV irradiance in W/m^2
// from the broadband direct and diffuse downward shortwave components at the
// surface (both in W/m^2). Returns NaN if either input is NaN or the sum is
// negative.
func ErythemalUVFromSW(directSW, diffuseSW float64) float64 {
	if math.IsNaN(directSW) || math.IsNaN(diffuseSW) {
		return math.NaN()
	}
	sw := directSW + diffuseSW
	if sw < 0 {
		return math.NaN()
	}
	return erythemalUVFraction * sw
}

// UVIndexFromErythemalUV converts an erythemal UV irradiance (W/m^2) to UV
// index. Returns NaN for NaN input.
func UVIndexFromErythemalUV(eryUV float64) float64 {
	if math.IsNaN(eryUV) {
		return math.NaN()
	}
	return eryUV * uviPerErythemalWm2
}

// UVDoseSEDStep returns the erythemal UV dose accumulated over durationS
// seconds at a constant erythemal UV irradiance eryUV (W/m^2), expressed in
// Standard Erythema Doses (SED). Returns 0 for NaN irradiance or
// non-positive duration so it is safe to use in a running sum.
func UVDoseSEDStep(eryUV, durationS float64) float64 {
	if math.IsNaN(eryUV) || durationS <= 0 || eryUV <= 0 {
		return 0
	}
	return eryUV * durationS / sedJoulesPerM2
}
