package sg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
)

func TestNullTime(t *testing.T) {
	nt := nullTime(time.Time{})
	require.False(t, nt.Valid)

	now := time.Now()
	nt = nullTime(now)
	require.True(t, nt.Valid)
	require.Equal(t, now, nt.Time)
}

func TestParseDateTime(t *testing.T) {
	cases := []struct {
		in      string
		ok      bool
		y, m, d int
	}{
		{"2026-04-14 00:00:00+00:00", true, 2026, 4, 14},
		{"2026-06-26 00:00:00+00:00", true, 2026, 6, 26},
		{"2025-12-31 23:59:59+01:00", true, 2025, 12, 31},
		{"", false, 0, 0, 0},
		{"undefiniert", false, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseDateTime(tc.in)
			if tc.ok {
				require.NoError(t, err)
				require.Equal(t, tc.y, got.Year())
				require.Equal(t, time.Month(tc.m), got.Month())
				require.Equal(t, tc.d, got.Day())
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestDecodeFeature exercises the full decode path with a payload shaped like
// the live SG WFS response. Fixture values are derived from real Canton
// St. Gallen open data (CC BY 4.0; see testdata/LICENSE).
func TestDecodeFeature(t *testing.T) {
	innerXML := []byte(`
		<ods:geo_point_2d xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype"
		                  xmlns:gml="http://www.opengis.net/gml/3.2">
		  <gml:Point srsName="urn:ogc:def:crs:EPSG::4326">
		    <gml:pos>47.4258329 9.0876304</gml:pos>
		  </gml:Point>
		</ods:geo_point_2d>
		<ods:geo_shape xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype"
		               xmlns:gml="http://www.opengis.net/gml/3.2">
		  <gml:Point srsName="urn:ogc:def:crs:EPSG::4326">
		    <gml:pos>47.4258329 9.0876304</gml:pos>
		  </gml:Point>
		</ods:geo_shape>
		<ods:id xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype">29082</ods:id>
		<ods:bew xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype">Jonschwil Poststrasse</ods:bew>
		<ods:zust xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype">Tiefbauamt Kanton St.Gallen</ods:zust>
		<ods:adresse xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype">Poststrasse 10, 9243 Jonschwil, Schweiz</ods:adresse>
		<ods:beginn xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype">2026-04-15 00:00:00+00:00</ods:beginn>
		<ods:ende xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype">2026-06-26 00:00:00+00:00</ods:ende>
	`)

	f, err := decodeFeature(wfs.Feature{InnerXML: innerXML})
	require.NoError(t, err)

	require.Equal(t, "sg-29082", f.SourceID)
	require.Equal(t, "Jonschwil Poststrasse", f.Bew)
	require.Equal(t, "Tiefbauamt Kanton St.Gallen", f.Zust)
	require.Equal(t, "Poststrasse 10, 9243 Jonschwil, Schweiz", f.Adresse)

	require.Equal(t, 2026, f.Beginn.Year())
	require.Equal(t, time.April, f.Beginn.Month())
	require.Equal(t, 15, f.Beginn.Day())

	require.Equal(t, 2026, f.Ende.Year())
	require.Equal(t, time.June, f.Ende.Month())
	require.Equal(t, 26, f.Ende.Day())

	require.NotNil(t, f.Geometry)

	// Geometry must be in (lon, lat) order, swapped from GML EPSG:4326 axis order.
	g := f.Geometry.Geometry()
	require.NotNil(t, g)
	bound := g.Bound()
	// Longitude should be in Swiss eastern range (~8-10 deg E).
	require.InDelta(t, 9.0876304, bound.Min.Lon(), 1e-6)
	// Latitude should be in Swiss range (~46-48 deg N).
	require.InDelta(t, 47.4258329, bound.Min.Lat(), 1e-6)
}

// TestDecodeFeatureMissingDates verifies that a feature with absent date
// fields decodes without error and leaves date fields at their zero value.
func TestDecodeFeatureMissingDates(t *testing.T) {
	innerXML := []byte(`
		<ods:geo_shape xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype"
		               xmlns:gml="http://www.opengis.net/gml/3.2">
		  <gml:Point srsName="urn:ogc:def:crs:EPSG::4326">
		    <gml:pos>47.0 9.0</gml:pos>
		  </gml:Point>
		</ods:geo_shape>
		<ods:id xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype">99999</ods:id>
		<ods:bew xmlns:ods="https://stgallen.opendatasoft.com/api/wfs/featuretype">Test</ods:bew>
	`)
	f, err := decodeFeature(wfs.Feature{InnerXML: innerXML})
	require.NoError(t, err)
	require.Equal(t, "sg-99999", f.SourceID)
	require.True(t, f.Beginn.IsZero())
	require.True(t, f.Ende.IsZero())
	require.NotNil(t, f.Geometry)
}
