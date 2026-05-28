package tg

import (
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/require"
)

func TestSourceID(t *testing.T) {
	require.Equal(t, "tg-5331", sourceID("5331"))
	require.Equal(t, "tg-", sourceID(""))
}

func TestFeatureTitle(t *testing.T) {
	// Project name wins when present.
	got := featureTitle(Properties{
		Projektbezeichnung: "Sanierung Bushaltestelle Seeblick",
		Projektnummer:      "4431-128",
	})
	require.Equal(t, "Sanierung Bushaltestelle Seeblick", got)

	// Project number is used when no project name is provided.
	got = featureTitle(Properties{Projektnummer: "4431-128"})
	require.Equal(t, "4431-128", got)

	// Both empty yields an empty title (caller's responsibility to handle).
	require.Empty(t, featureTitle(Properties{}))
}

func TestParseDate(t *testing.T) {
	got := parseDate("2026-03-16")
	require.True(t, got.Valid)
	require.Equal(t, time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC), got.Time)

	require.False(t, parseDate("").Valid)
	require.False(t, parseDate("16.03.2026").Valid)
	require.False(t, parseDate("not a date").Valid)
}

func TestReprojectGeometryNil(t *testing.T) {
	require.Nil(t, reprojectGeometry(nil))
}

func TestReprojectGeometryLineString(t *testing.T) {
	// LV95 coordinates near Bern federal palace, taken from the swisstopo
	// PDF worked example: (2600000, 1200000) -> approx (7.43863, 46.95108).
	src := geojson.NewGeometry(orb.LineString{
		{2600000, 1200000},
		{2700000, 1100000},
	})
	out := reprojectGeometry(src)
	require.NotNil(t, out)

	ls, ok := out.Geometry().(orb.LineString)
	require.True(t, ok, "expected LineString, got %T", out.Geometry())
	require.Len(t, ls, 2)

	// Bern reference point.
	require.InDelta(t, 7.43863, ls[0][0], 1e-4)
	require.InDelta(t, 46.95108, ls[0][1], 1e-4)

	// PDF worked example: (2700000, 1100000) -> ~ (8.7305, 46.04413).
	require.InDelta(t, 8.7305, ls[1][0], 1e-3)
	require.InDelta(t, 46.04413, ls[1][1], 1e-3)
}

func TestReprojectGeometryDoesNotMutateInput(t *testing.T) {
	src := geojson.NewGeometry(orb.LineString{{2600000, 1200000}})
	_ = reprojectGeometry(src)
	// Original geometry remains in LV95.
	ls := src.Geometry().(orb.LineString)
	require.Equal(t, 2600000.0, ls[0][0])
	require.Equal(t, 1200000.0, ls[0][1])
}

func TestGetURLContainsRequiredParams(t *testing.T) {
	u := getURL()
	require.Contains(t, u, "layers=all%3Abaustellen-baustelle")
	require.Contains(t, u, "geometryType=esriGeometryEnvelope")
	require.Contains(t, u, "geometryFormat=geojson")
	require.Contains(t, u, "returnGeometry=true")
	require.Contains(t, u, "tolerance=0")
}
