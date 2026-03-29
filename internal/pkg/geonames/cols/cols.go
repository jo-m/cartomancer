// Package cols contains GeoNames column definitions parsed from the official readme.
package cols

//go:generate go run gen_cols.go
//go:generate go run gen_feature_codes.go

// BaseURL is the GeoNames data dump base URL, without trailing slash.
const BaseURL = "https://download.geonames.org/export/dump"

// Column describes a single column in the GeoNames main table.
type Column struct {
	// Index is the zero-based position in the tab-delimited row.
	Index int
	// Name is the original column name from the readme (e.g. "geonameid").
	Name string
	// GoName is the Go-friendly identifier (e.g. "Geonameid").
	GoName string
	// Description is the human-readable description from the readme.
	Description string
}
