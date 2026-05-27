package ag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/paulmach/orb/geojson"
)

// FeatureCollection is the top-level GeoJSON response from the AG
// ArcGIS MapServer query endpoint.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

// Feature is a single construction-site record returned by the AG MapServer.
type Feature struct {
	Type       string            `json:"type"`
	ID         int64             `json:"id"`
	Geometry   *geojson.Geometry `json:"geometry"`
	Properties Properties        `json:"properties"`
}

// Properties holds the schema-specific fields of one AG construction-site
// feature. Field names mirror the upstream attribute table on layer 0 of
// the ATB/Baustellen_online MapServer; only the fields consumed by this
// package are declared. Date fields decode via [apiDate].
type Properties struct {
	// ObjectID is the stable primary key of the feature.
	ObjectID int64 `json:"OBJECTID"`

	// PSCode is the project code assigned by the canton.
	PSCode string `json:"PSCode"`

	// Achsen names the affected road axis or axes.
	Achsen string `json:"Achsen"`

	// Gemeinde is the host municipality.
	Gemeinde string `json:"Gemeinde"`

	// Bezeichnung is the human-readable project description.
	Bezeichnung string `json:"Bezeichnung"`

	// Bauherr names the client / owner of the works.
	Bauherr string `json:"Bauherr"`

	// BehinderungKarte is the impairment description shown in the map popup.
	BehinderungKarte string `json:"Behinderung_Karte"`

	// BehinderungTabelle is the impairment description shown in the table view.
	BehinderungTabelle string `json:"Behinderung_Tabelle"`

	// FDate is the active-from date of the site.
	FDate apiDate `json:"fDate"`

	// TDate is the active-to date of the site.
	TDate apiDate `json:"tDate"`
}

// apiDate decodes the AG MapServer date encoding. The upstream service
// returns dates as ISO-8601 strings when queried with f=geojson and as
// Unix epoch milliseconds when queried with f=json; both forms are
// accepted to stay robust against future server changes.
type apiDate struct {
	// Time is the parsed timestamp. Zero when the source value was null,
	// empty, or in an unrecognised format.
	Time time.Time
}

// dateLayouts lists the string layouts accepted by [apiDate.UnmarshalJSON].
// They are tried in order; the first match wins.
var dateLayouts = []string{
	"2006-01-02T15:04:05.000Z",
	"2006-01-02T15:04:05Z",
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// UnmarshalJSON decodes either a JSON number (Unix epoch milliseconds) or
// a JSON string in one of the layouts listed in [dateLayouts] into d.
// A null or empty value leaves d at its zero value and returns no error.
func (d *apiDate) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}

	if data[0] != '"' {
		var ms int64
		if err := json.Unmarshal(data, &ms); err != nil {
			return fmt.Errorf("decode numeric date: %w", err)
		}
		d.Time = time.UnixMilli(ms).UTC()
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("decode string date: %w", err)
	}
	if s == "" {
		return nil
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			d.Time = t
			return nil
		}
	}
	return fmt.Errorf("unrecognised date format: %q", s)
}
