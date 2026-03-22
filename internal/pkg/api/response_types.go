package api

import "database/sql"

// lonLatResponse represents a geographic coordinate pair.
type lonLatResponse struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// nullLonLat returns a lonLatResponse pointer from nullable lon/lat values.
// Returns nil if either value is null.
func nullLonLat(lon, lat sql.NullFloat64) *lonLatResponse {
	if !lon.Valid || !lat.Valid {
		return nil
	}
	return &lonLatResponse{Lat: lat.Float64, Lon: lon.Float64}
}

// nullBBox returns a bboxResponse pointer from four nullable bound values.
// Returns nil if any value is null.
func nullBBox(minLat, minLon, maxLat, maxLon sql.NullFloat64) *bboxResponse {
	if !minLat.Valid || !minLon.Valid || !maxLat.Valid || !maxLon.Valid {
		return nil
	}
	return &bboxResponse{
		Min: lonLatResponse{Lat: minLat.Float64, Lon: minLon.Float64},
		Max: lonLatResponse{Lat: maxLat.Float64, Lon: maxLon.Float64},
	}
}

// nullMinMax returns a minMaxResponse pointer from two nullable float64 values.
// Returns nil if both values are null.
func nullMinMax(min, max sql.NullFloat64) *minMaxResponse {
	if !min.Valid && !max.Valid {
		return nil
	}
	return &minMaxResponse{
		Min: nullFloat64Ptr(min),
		Max: nullFloat64Ptr(max),
	}
}

// bboxResponse represents a geographic bounding box defined by min/max corners.
type bboxResponse struct {
	Min lonLatResponse `json:"min"`
	Max lonLatResponse `json:"max"`
}

// minMaxResponse represents a nullable min/max range of float64 values.
type minMaxResponse struct {
	Min *float64 `json:"min"`
	Max *float64 `json:"max"`
}

// userRefResponse is a lightweight user reference containing only UUID and display name.
type userRefResponse struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// attributionResponse groups a human-readable attribution with its source URL.
type attributionResponse struct {
	Text string `json:"text"`
	Href string `json:"href"`
}
