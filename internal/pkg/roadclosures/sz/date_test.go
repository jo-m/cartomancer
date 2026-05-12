package sz

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseGermanDate(t *testing.T) {
	cases := []struct {
		in      string
		ok      bool
		y, m, d int
	}{
		{"13. April 2026", true, 2026, 4, 13},
		{"16. März 2026", true, 2026, 3, 16},
		{"16. Maerz 2026", true, 2026, 3, 16},
		{"2. Februar 2026", true, 2026, 2, 2},
		{"03. Juni 2026", true, 2026, 6, 3},
		{"18.11.2024", true, 2024, 11, 18},
		{"31.07.2026", true, 2026, 7, 31},
		{"5.3.2025", true, 2025, 3, 5},
		{"ca. 30. November 2026", true, 2026, 11, 30},
		{"  30. November 2026 ", true, 2026, 11, 30},
		{"", false, 0, 0, 0},
		{"undefiniert", false, 0, 0, 0},
		{"32. April 2026", false, 0, 0, 0},
		{"15. Foo 2026", false, 0, 0, 0},
		{"15. April 1800", false, 0, 0, 0},
	}
	for _, tc := range cases {
		got, ok := parseGermanDate(tc.in)
		require.Equal(t, tc.ok, ok, "input=%q", tc.in)
		if !tc.ok {
			continue
		}
		require.Equal(t, tc.y, got.Year(), "input=%q", tc.in)
		require.Equal(t, time.Month(tc.m), got.Month(), "input=%q", tc.in)
		require.Equal(t, tc.d, got.Day(), "input=%q", tc.in)
		require.Equal(t, time.UTC, got.Location(), "input=%q", tc.in)
	}
}
