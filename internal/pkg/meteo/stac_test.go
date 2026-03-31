package meteo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/geoadmin"
)

func TestNewestReferenceTime(t *testing.T) {
	tests := []struct {
		name     string
		coll     geoadmin.Collection
		wantZero bool
	}{
		{
			name:     "empty interval",
			coll:     geoadmin.Collection{},
			wantZero: true,
		},
		{
			name: "valid interval",
			coll: geoadmin.Collection{
				Extent: geoadmin.Extent{
					Temporal: geoadmin.TemporalExtent{
						Interval: [][2]time.Time{
							{
								time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
								time.Date(2026, 3, 11, 9, 0, 0, 0, time.UTC),
							},
						},
					},
				},
			},
			wantZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newestReferenceTime(&tt.coll)
			if tt.wantZero {
				require.True(t, got.IsZero())
			} else {
				require.False(t, got.IsZero())
				// 2026-03-11T09:00:00Z - 33h = 2026-03-10T00:00:00Z
				require.Equal(t, time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), got)
			}
		})
	}
}
