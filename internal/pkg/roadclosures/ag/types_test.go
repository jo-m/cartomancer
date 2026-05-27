package ag

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApiDate_UnmarshalString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		ok   bool
		y    int
		m    time.Month
		d    int
	}{
		{"rfc3339_z", `"2025-11-19T00:00:00Z"`, true, 2025, time.November, 19},
		{"esri_millis_z", `"2025-11-19T00:00:00.000Z"`, true, 2025, time.November, 19},
		{"no_tz", `"2026-05-01T12:34:56"`, true, 2026, time.May, 1},
		{"date_only", `"2026-05-01"`, true, 2026, time.May, 1},
		{"null", `null`, true, 0, 0, 0},
		{"empty_string", `""`, true, 0, 0, 0},
		{"garbage", `"not a date"`, false, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var d apiDate
			err := json.Unmarshal([]byte(tc.in), &d)
			if !tc.ok {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.y == 0 {
				require.True(t, d.Time.IsZero())
				return
			}
			require.Equal(t, tc.y, d.Time.Year())
			require.Equal(t, tc.m, d.Time.Month())
			require.Equal(t, tc.d, d.Time.Day())
		})
	}
}

func TestApiDate_UnmarshalNumeric(t *testing.T) {
	// 2026-05-27T00:00:00Z in epoch milliseconds.
	ms := time.Date(2026, time.May, 27, 0, 0, 0, 0, time.UTC).UnixMilli()
	in := []byte(`{"fDate":` + jsonNumber(ms) + `}`)
	var p Properties
	require.NoError(t, json.Unmarshal(in, &p))
	require.Equal(t, 2026, p.FDate.Time.Year())
	require.Equal(t, time.May, p.FDate.Time.Month())
	require.Equal(t, 27, p.FDate.Time.Day())
}

func TestDecodeFeatureCollection(t *testing.T) {
	payload := []byte(`{
	  "type": "FeatureCollection",
	  "features": [
	    {
	      "type": "Feature",
	      "id": 4711,
	      "geometry": {
	        "type": "Polygon",
	        "coordinates": [[[8.04,47.39],[8.05,47.39],[8.05,47.40],[8.04,47.40],[8.04,47.39]]]
	      },
	      "properties": {
	        "OBJECTID": 4711,
	        "PSCode": "K123",
	        "Achsen": "K123 Aarau - Lenzburg",
	        "Gemeinde": "Aarau",
	        "Bezeichnung": "Sanierung Hauptstrasse",
	        "Bauherr": "Kanton Aargau",
	        "Behinderung_Karte": "Vollsperrung mit Umleitung",
	        "Behinderung_Tabelle": "Strasse gesperrt",
	        "fDate": "2026-04-15T00:00:00Z",
	        "tDate": "2026-06-26T00:00:00Z"
	      }
	    }
	  ]
	}`)

	var fc FeatureCollection
	require.NoError(t, json.Unmarshal(payload, &fc))
	require.Equal(t, "FeatureCollection", fc.Type)
	require.Len(t, fc.Features, 1)

	f := fc.Features[0]
	require.Equal(t, int64(4711), f.Properties.ObjectID)
	require.Equal(t, "Sanierung Hauptstrasse", f.Properties.Bezeichnung)
	require.Equal(t, "Vollsperrung mit Umleitung", f.Properties.BehinderungKarte)
	require.NotNil(t, f.Geometry)
	require.Equal(t, "Polygon", f.Geometry.Type)
	require.Equal(t, 2026, f.Properties.FDate.Time.Year())
	require.Equal(t, time.April, f.Properties.FDate.Time.Month())
	require.Equal(t, 15, f.Properties.FDate.Time.Day())
	require.Equal(t, time.June, f.Properties.TDate.Time.Month())
}

// jsonNumber renders an int64 as a string usable inside a JSON literal.
func jsonNumber(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
