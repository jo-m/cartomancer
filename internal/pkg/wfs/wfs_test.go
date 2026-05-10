package wfs

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return b
}

func TestParseCapabilities(t *testing.T) {
	body := readFixture(t, "capabilities.xml")

	var caps Capabilities
	require.NoError(t, xml.Unmarshal(body, &caps))

	require.Equal(t, "2.0.0", caps.Version)
	require.Equal(t, "Baustellen", caps.ServiceIdentification.Title)
	require.Equal(t, "Geodienst des GIS-ZH", caps.ServiceIdentification.Abstract)
	require.Equal(t, "WFS", caps.ServiceIdentification.ServiceType)
	require.Equal(t, "2.0.0", caps.ServiceIdentification.ServiceTypeVersion)

	require.Len(t, caps.FeatureTypes, 2)

	ft := caps.FeatureTypes[0]
	require.Equal(t, "ms:baustellen-detailansicht", ft.Name)
	require.Equal(t, "Baustellen Detailansicht", ft.Title)
	require.Equal(t, "urn:ogc:def:crs:EPSG::2056", ft.DefaultCRS)
	require.Contains(t, ft.OtherCRS, "urn:ogc:def:crs:EPSG::4326")
	require.Contains(t, ft.OutputFormats, "application/json; subtype=geojson")
	require.Equal(t, "8.157224 47.141694", ft.WGS84BoundingBox.LowerCorner)
	require.Equal(t, "9.037478 47.712435", ft.WGS84BoundingBox.UpperCorner)
	require.Equal(t,
		"https://maps.zh.ch/wfs/TbaBaustellenZHWFS?request=GetMetadata&layer=baustellen-detailansicht",
		ft.MetadataURL.Href,
	)

	require.Equal(t, "ms:baustellen-uebersicht", caps.FeatureTypes[1].Name)
}

func TestParseFeatureCollection(t *testing.T) {
	body := readFixture(t, "features.xml")

	var fc FeatureCollection
	require.NoError(t, xml.Unmarshal(body, &fc))

	require.Equal(t, "2026-05-10T22:00:12", fc.TimeStamp)
	require.Equal(t, "unknown", fc.NumberMatched)
	require.Equal(t, 3, fc.NumberReturned)
	require.Equal(t,
		"https://maps.zh.ch/wfs/TbaBaustellenZHWFS?service=WFS&version=2.0.0&request=GetFeature&typeNames=ms%3Abaustellen-uebersicht&count=3&STARTINDEX=3",
		fc.Next,
	)
	require.Len(t, fc.Members, 3)

	first := fc.Members[0].Feature
	require.Equal(t, "http://mapserver.gis.umn.edu/mapserver", first.XMLName.Space)
	require.Equal(t, "baustellen-uebersicht", first.XMLName.Local)
	require.Equal(t, "baustellen-uebersicht.2744", first.GMLID)
	// The inner XML must preserve property children so callers can re-parse
	// schema-specific fields themselves.
	require.Contains(t, string(first.InnerXML), "<ms:strassenname>Seestrasse")
	require.Contains(t, string(first.InnerXML), "<gml:Point")
}

func TestParseFeatureCollectionEmpty(t *testing.T) {
	body := readFixture(t, "features_empty.xml")

	var fc FeatureCollection
	require.NoError(t, xml.Unmarshal(body, &fc))

	require.Equal(t, 0, fc.NumberReturned)
	require.Empty(t, fc.Members)
	require.Empty(t, fc.Next)
}

func TestParseException(t *testing.T) {
	body := readFixture(t, "exception.xml")

	exc := parseException(body)
	require.NotNil(t, exc)
	require.Equal(t, "2.0.0", exc.Version)
	require.Len(t, exc.Exceptions, 1)
	require.Equal(t, "InvalidParameterValue", exc.Exceptions[0].Code)
	require.Equal(t, "typename", exc.Exceptions[0].Locator)
	require.Contains(t, exc.Exceptions[0].Texts[0], "doesn't exist in this server")

	// As an error, the formatted message must include code, locator and text.
	msg := exc.Error()
	require.Contains(t, msg, "InvalidParameterValue")
	require.Contains(t, msg, "typename")
	require.Contains(t, msg, "doesn't exist")
}

func TestParseExceptionIgnoresNonException(t *testing.T) {
	require.Nil(t, parseException(readFixture(t, "capabilities.xml")))
	require.Nil(t, parseException(readFixture(t, "features.xml")))
	require.Nil(t, parseException(readFixture(t, "features_empty.xml")))
}

func TestNewClientTrimsTrailing(t *testing.T) {
	require.Equal(t, "https://example.com/wfs", NewClient("https://example.com/wfs").baseURL)
	require.Equal(t, "https://example.com/wfs", NewClient("https://example.com/wfs/").baseURL)
	require.Equal(t, "https://example.com/wfs", NewClient("https://example.com/wfs?").baseURL)
}

func TestRequestURLAddsServiceAndVersion(t *testing.T) {
	c := NewClient("https://example.com/wfs")
	u := c.requestURL(map[string][]string{"request": {"GetCapabilities"}})
	// url.Values.Encode sorts keys alphabetically, so the order is stable.
	require.Equal(t,
		"https://example.com/wfs?request=GetCapabilities&service=WFS&version=2.0.0",
		u,
	)
}
