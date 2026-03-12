package params

// Param identifies a meteorological variable by its GRIB2 discipline, category,
// and parameter codes (cf. WMO Manual on Codes, GRIB2 Table 4.2 and local
// MeteoSwiss extensions).
type Param [3]uint8
