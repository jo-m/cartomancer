package rest

import (
	"net/http"
	"strconv"

	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

const (
	// defaultSearchRadiusDeg is the default bounding box half-width in degrees
	// (~50 km at mid-latitudes).
	defaultSearchRadiusDeg = 0.5

	// maxReverseGeocodeResults caps the number of results returned.
	maxReverseGeocodeResults = 5
)

type reverseGeocodeResult struct {
	Geonameid    int64   `json:"geonameid"`
	Name         string  `json:"name"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	FeatureClass string  `json:"featureClass"`
	FeatureCode  string  `json:"featureCode"`
	CountryCode  string  `json:"countryCode"`
	Admin1Code   string  `json:"admin1Code"`
}

type reverseGeocodeResponse struct {
	Results []reverseGeocodeResult `json:"results"`
}

// handleReverseGeocode returns the nearest populated places for a lat/lon pair.
// Query params: lat (float64), lon (float64), limit (optional int, default 1, max 5).
func (sv *server) handleReverseGeocode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	latStr := q.Get("lat")
	lonStr := q.Get("lon")
	if latStr == "" || lonStr == "" {
		writeError(w, http.StatusBadRequest, "lat and lon are required")
		return
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		writeError(w, http.StatusBadRequest, "lat must be a number between -90 and 90")
		return
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil || lon < -180 || lon > 180 {
		writeError(w, http.StatusBadRequest, "lon must be a number between -180 and 180")
		return
	}

	limit := int64(1)
	if limitStr := q.Get("limit"); limitStr != "" {
		v, err := strconv.ParseInt(limitStr, 10, 64)
		if err != nil || v < 1 || v > maxReverseGeocodeResults {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 5")
			return
		}
		limit = v
	}

	rows, err := sv.d.QueryRO().ReverseGeocode(ctx, db.ReverseGeocodeParams{
		Lat:        lat,
		Lon:        lon,
		MinLat:     lat - defaultSearchRadiusDeg,
		MaxLat:     lat + defaultSearchRadiusDeg,
		MinLon:     lon - defaultSearchRadiusDeg,
		MaxLon:     lon + defaultSearchRadiusDeg,
		MaxResults: limit,
	})
	if err != nil {
		logg.Error(ctx, "reverse geocode query failed", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	results := make([]reverseGeocodeResult, len(rows))
	for i, row := range rows {
		results[i] = reverseGeocodeResult{
			Geonameid:    row.Geonameid,
			Name:         row.Name,
			Latitude:     row.Latitude,
			Longitude:    row.Longitude,
			FeatureClass: row.FeatureClass,
			FeatureCode:  row.FeatureCode,
			CountryCode:  row.CountryCode,
			Admin1Code:   row.Admin1Code,
		}
	}

	writeJSON(w, http.StatusOK, reverseGeocodeResponse{Results: results})
}
