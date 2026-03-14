package stac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseISO8601Duration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "hours only",
			input: "PT1H",
			want:  time.Hour,
		},
		{
			name:  "minutes only",
			input: "PT30M",
			want:  30 * time.Minute,
		},
		{
			name:  "seconds only",
			input: "PT45S",
			want:  45 * time.Second,
		},
		{
			name:  "days only",
			input: "P2DT",
			want:  48 * time.Hour,
		},
		{
			name:  "hours and minutes",
			input: "PT1H30M",
			want:  time.Hour + 30*time.Minute,
		},
		{
			name:  "days hours minutes seconds",
			input: "P1DT2H3M4S",
			want:  24*time.Hour + 2*time.Hour + 3*time.Minute + 4*time.Second,
		},
		{
			name:  "zero duration",
			input: "PT0S",
			want:  0,
		},
		{
			name:  "large values",
			input: "P30DT23H59M59S",
			want:  30*24*time.Hour + 23*time.Hour + 59*time.Minute + 59*time.Second,
		},
		{
			name:  "33 hours as used in CollectionHorizon",
			input: "P1DT9H",
			want:  33 * time.Hour,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing T separator",
			input:   "P1H",
			wantErr: true,
		},
		{
			name:    "missing P prefix",
			input:   "T1H",
			wantErr: true,
		},
		{
			name:    "garbage input",
			input:   "not a duration",
			wantErr: true,
		},
		{
			name:    "year-month duration not supported",
			input:   "P1Y2M",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseISO8601Duration(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
