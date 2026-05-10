package zh

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
)

func TestIsActiveStatus(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"aktiv (Bauzeit)", true},
		{"zukünftig (Bauzeit in Zukunft)", true},
		{"  Aktiv ", true},
		{"abgeschlossen", false},
		{"", false},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, isActiveStatus(tc.in), "status=%q", tc.in)
	}
}

func TestFilterActive(t *testing.T) {
	in := []Feature{
		{GMLID: "a", StatusBaustelle: "aktiv (Bauzeit)"},
		{GMLID: "b", StatusBaustelle: "zukünftig (Bauzeit in Zukunft)"},
		{GMLID: "c", StatusBaustelle: "abgeschlossen"},
		{GMLID: "d", StatusBaustelle: ""},
	}
	out := filterActive(in)
	require.Len(t, out, 2)
	require.Equal(t, "a", out[0].GMLID)
	require.Equal(t, "b", out[1].GMLID)
}

func TestNullTime(t *testing.T) {
	nt := nullTime(time.Time{})
	require.False(t, nt.Valid)

	now := time.Now()
	nt = nullTime(now)
	require.True(t, nt.Valid)
	require.Equal(t, now, nt.Time)
}

func TestDecodeFeature(t *testing.T) {
	innerXML := []byte(`
        <gml:boundedBy xmlns:gml="http://www.opengis.net/gml/3.2">
          <gml:Envelope srsName="urn:ogc:def:crs:EPSG::4326">
            <gml:lowerCorner>47.226451 8.673930</gml:lowerCorner>
            <gml:upperCorner>47.229651 8.677622</gml:upperCorner>
          </gml:Envelope>
        </gml:boundedBy>
        <ms:geometry xmlns:ms="http://mapserver.gis.umn.edu/mapserver" xmlns:gml="http://www.opengis.net/gml/3.2">
          <gml:Point srsName="urn:ogc:def:crs:EPSG::4326"><gml:pos>47.226 8.677</gml:pos></gml:Point>
        </ms:geometry>
        <ms:strassenname xmlns:ms="http://mapserver.gis.umn.edu/mapserver">Seestrasse</ms:strassenname>
        <ms:gemeindename xmlns:ms="http://mapserver.gis.umn.edu/mapserver">Wädenswil</ms:gemeindename>
        <ms:beschreibung xmlns:ms="http://mapserver.gis.umn.edu/mapserver">Werleitungsarbeiten</ms:beschreibung>
        <ms:verkehrsfuehrung xmlns:ms="http://mapserver.gis.umn.edu/mapserver">Einbahnverkehr</ms:verkehrsfuehrung>
        <ms:status_baustelle xmlns:ms="http://mapserver.gis.umn.edu/mapserver">aktiv (Bauzeit)</ms:status_baustelle>
        <ms:datum_baubeginn xmlns:ms="http://mapserver.gis.umn.edu/mapserver" xmlns:gml="http://www.opengis.net/gml/3.2"><gml:timePosition>2026-06-01T14:22:18Z</gml:timePosition></ms:datum_baubeginn>
        <ms:datum_bauende xmlns:ms="http://mapserver.gis.umn.edu/mapserver" xmlns:gml="http://www.opengis.net/gml/3.2"><gml:timePosition>2026-11-20T15:22:34Z</gml:timePosition></ms:datum_bauende>
    `)
	f, err := decodeFeature(wfs.Feature{GMLID: "baustellen-detailansicht.2744", InnerXML: innerXML})
	require.NoError(t, err)
	require.Equal(t, "baustellen-detailansicht.2744", f.GMLID)
	require.Equal(t, "Seestrasse", f.Strassenname)
	require.Equal(t, "Wädenswil", f.Gemeindename)
	require.Equal(t, "Werleitungsarbeiten", f.Beschreibung)
	require.Equal(t, "Einbahnverkehr", f.Verkehrsfuehrung)
	require.Equal(t, "aktiv (Bauzeit)", f.StatusBaustelle)
	require.Equal(t, 2026, f.DatumBaubeginn.Year())
	require.Equal(t, time.June, f.DatumBaubeginn.Month())
	require.Equal(t, 2026, f.DatumBauende.Year())
	require.Equal(t, time.November, f.DatumBauende.Month())
	require.NotNil(t, f.Geometry)
}
