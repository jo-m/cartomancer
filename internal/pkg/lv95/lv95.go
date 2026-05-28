// Package lv95 converts Swiss LV95 (EPSG:2056) coordinates to WGS84 using the
// approximation formulas published by swisstopo. Accuracy is on the order of
// one meter, which is fine for road closure geometries matched against GPS tracks.
//
// Source: "Approximate formulas for the transformation between Swiss projection
// coordinates and WGS84", swisstopo, December 2016, section 2.
package lv95

// arcSecondsPerDegree scales from units of 10000 arc-seconds (used internally
// by the swisstopo polynomials) to decimal degrees: 1 degree = 3600 arc-seconds,
// so 10000 arc-seconds = 10000/3600 = 100/36 degrees.
const arcSecondsPerDegree = 100.0 / 36.0

// ToWGS84 converts a single LV95 easting/northing in meters to WGS84
// longitude/latitude in degrees using the swisstopo approximate formulas.
// The easting is the larger value (~2.6e6) and the northing is the smaller
// one (~1.2e6). The returned order is longitude first, then latitude, to
// match the GeoJSON axis convention.
func ToWGS84(easting, northing float64) (lon, lat float64) {
	// Shift to the civilian system (Bern = 0) and scale to units of 1000 km.
	y := (easting - 2600000.0) / 1000000.0
	x := (northing - 1200000.0) / 1000000.0

	lambda := 2.6779094 +
		4.728982*y +
		0.791484*y*x +
		0.1306*y*x*x -
		0.0436*y*y*y

	phi := 16.9023892 +
		3.238272*x -
		0.270978*y*y -
		0.002528*x*x -
		0.0447*y*y*x -
		0.0140*x*x*x

	return lambda * arcSecondsPerDegree, phi * arcSecondsPerDegree
}
