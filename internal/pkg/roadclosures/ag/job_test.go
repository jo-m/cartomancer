package ag

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures"
)

func TestSourceID(t *testing.T) {
	require.Equal(t, "ag-42", sourceID(42))
	require.Equal(t, "ag-0", sourceID(0))
}

func TestFeatureTitle(t *testing.T) {
	got := featureTitle(Properties{Bezeichnung: "Sanierung", Gemeinde: "Aarau"})
	require.Equal(t, "Aarau - Sanierung", got)
}

func TestFeatureDescription(t *testing.T) {
	// Returns BehinderungTabelle only; other fields are ignored.
	got := featureDescription(Properties{
		BehinderungKarte:   "Vollsperrung",
		BehinderungTabelle: "Strasse gesperrt",
		Achsen:             "K123",
		Gemeinde:           "Aarau",
	})
	require.Equal(t, "Strasse gesperrt", got)
}

func TestClosureTypeFromText(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want roadclosures.ClosureType
	}{
		{"vollsperrung_title", []string{"Vollsperrung Hauptstrasse", "", ""}, roadclosures.ClosedWay},
		{"gesperrt_lowercase", []string{"Baustelle", "strasse gesperrt", ""}, roadclosures.ClosedWay},
		{"sperrung_mixed_case", []string{"SPERRUNG", "", ""}, roadclosures.ClosedWay},
		{"detour_default", []string{"Umleitung Kantonsstrasse", "", ""}, roadclosures.Detour},
		{"all_empty", []string{"", "", ""}, roadclosures.Detour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, closureTypeFromText(tc.in...))
		})
	}
}

func TestNullTime(t *testing.T) {
	nt := nullTime(time.Time{})
	require.False(t, nt.Valid)

	now := time.Now()
	nt = nullTime(now)
	require.True(t, nt.Valid)
	require.Equal(t, now, nt.Time)
}
