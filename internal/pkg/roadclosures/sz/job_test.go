package sz

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

func TestBuildDescription(t *testing.T) {
	cases := []struct {
		name string
		in   Feature
		want string
	}{
		{
			name: "both present",
			in:   Feature{Behinderungsbemerkung: "Einbahnverkehr", Link: "https://example.test/x"},
			want: "Einbahnverkehr\n\nhttps://example.test/x",
		},
		{
			name: "only note",
			in:   Feature{Behinderungsbemerkung: "Strasse gesperrt"},
			want: "Strasse gesperrt",
		},
		{
			name: "only link",
			in:   Feature{Link: "https://example.test/y"},
			want: "https://example.test/y",
		},
		{
			name: "neither",
			in:   Feature{},
			want: "",
		},
		{
			name: "whitespace stripped",
			in:   Feature{Behinderungsbemerkung: "  ", Link: " https://example.test/z "},
			want: "https://example.test/z",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, buildDescription(tc.in))
		})
	}
}

// TestDecodeFeature exercises the full decode path with a payload shaped
// like the live SZ WFS response (LineString geometry, German date strings,
// no gml:id on the feature element).
func TestDecodeFeature(t *testing.T) {
	innerXML := []byte(`
        <gml:boundedBy xmlns:gml="http://www.opengis.net/gml/3.2">
          <gml:Envelope srsName="urn:ogc:def:crs:EPSG::4326">
            <gml:lowerCorner>47.023222 8.635973</gml:lowerCorner>
            <gml:upperCorner>47.024481 8.636844</gml:upperCorner>
          </gml:Envelope>
        </gml:boundedBy>
        <ms:geom xmlns:ms="http://mapserver.gis.umn.edu/mapserver" xmlns:gml="http://www.opengis.net/gml/3.2">
          <gml:LineString gml:id=".1" srsName="urn:ogc:def:crs:EPSG::4326">
            <gml:posList srsDimension="2">47.024481 8.635973 47.023222 8.636844</gml:posList>
          </gml:LineString>
        </ms:geom>
        <ms:lokalname xmlns:ms="http://mapserver.gis.umn.edu/mapserver">2 / Gotthardstrasse, Rösslimatt, Seewen</ms:lokalname>
        <ms:baubeginn_ui xmlns:ms="http://mapserver.gis.umn.edu/mapserver">13. April 2026</ms:baubeginn_ui>
        <ms:inbetriebnahme xmlns:ms="http://mapserver.gis.umn.edu/mapserver">20. Juni 2026</ms:inbetriebnahme>
        <ms:beschreibung xmlns:ms="http://mapserver.gis.umn.edu/mapserver">Behindertengerechter Ausbau der Bushaltestelle Rösslimatt in Seewen</ms:beschreibung>
        <ms:behinderungsbemerkung xmlns:ms="http://mapserver.gis.umn.edu/mapserver">mittel: Einspurig, mit LSA geregelt.</ms:behinderungsbemerkung>
        <ms:kontaktbauleitung xmlns:ms="http://mapserver.gis.umn.edu/mapserver">Joachim Gisler</ms:kontaktbauleitung>
        <ms:kontakttba xmlns:ms="http://mapserver.gis.umn.edu/mapserver">David Lüönd</ms:kontakttba>
        <ms:link xmlns:ms="http://mapserver.gis.umn.edu/mapserver"></ms:link>
    `)
	f, err := decodeFeature(wfs.Feature{InnerXML: innerXML}, 0)
	require.NoError(t, err)
	require.Equal(t, "2 / Gotthardstrasse, Rösslimatt, Seewen", f.Lokalname)
	require.Equal(t, "Behindertengerechter Ausbau der Bushaltestelle Rösslimatt in Seewen", f.Beschreibung)
	require.Equal(t, "mittel: Einspurig, mit LSA geregelt.", f.Behinderungsbemerkung)
	require.Equal(t, 2026, f.Baubeginn.Year())
	require.Equal(t, time.April, f.Baubeginn.Month())
	require.Equal(t, 13, f.Baubeginn.Day())
	require.Equal(t, 2026, f.Inbetriebnahme.Year())
	require.Equal(t, time.June, f.Inbetriebnahme.Month())
	require.NotNil(t, f.Geometry)
	require.NotEmpty(t, f.SourceID)

	// Geometry should be in (lon, lat) order, swapped from the source.
	pt, ok := firstPoint(f.Geometry)
	require.True(t, ok)
	require.InDelta(t, 8.635973, pt.Lon(), 1e-9)
	require.InDelta(t, 47.024481, pt.Lat(), 1e-9)
}

// TestSourceIDStable verifies that two decodes of the same feature produce
// the same SourceID, regardless of the within-cycle index (as long as the
// feature has a geometry to anchor the hash to).
func TestSourceIDStable(t *testing.T) {
	innerXML := []byte(`
        <ms:geom xmlns:ms="http://mapserver.gis.umn.edu/mapserver" xmlns:gml="http://www.opengis.net/gml/3.2">
          <gml:Point srsName="urn:ogc:def:crs:EPSG::4326"><gml:pos>47.0 8.5</gml:pos></gml:Point>
        </ms:geom>
        <ms:lokalname xmlns:ms="http://mapserver.gis.umn.edu/mapserver">Test</ms:lokalname>
    `)
	a, err := decodeFeature(wfs.Feature{InnerXML: innerXML}, 0)
	require.NoError(t, err)
	b, err := decodeFeature(wfs.Feature{InnerXML: innerXML}, 7)
	require.NoError(t, err)
	require.Equal(t, a.SourceID, b.SourceID)
}

// TestSourceIDDistinguishesGeomlessFeatures ensures that two geometry-less
// features with identical text fields still get unique IDs (via the index
// fallback) within a cycle.
func TestSourceIDDistinguishesGeomlessFeatures(t *testing.T) {
	innerXML := []byte(`<ms:lokalname xmlns:ms="http://mapserver.gis.umn.edu/mapserver">Same</ms:lokalname>`)
	a, err := decodeFeature(wfs.Feature{InnerXML: innerXML}, 0)
	require.NoError(t, err)
	b, err := decodeFeature(wfs.Feature{InnerXML: innerXML}, 1)
	require.NoError(t, err)
	require.NotEqual(t, a.SourceID, b.SourceID)
}
