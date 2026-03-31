//go:build online

package roadclosures_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures"
)

func TestOnlineFetchRoadClosures(t *testing.T) {
	ctx := context.Background()

	resp, err := roadclosures.Fetch(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Results, "expected at least one road closure result")

	for _, f := range resp.Results {
		require.Equal(t, "Feature", f.Type)
		require.NotZero(t, f.ID)
		require.NotZero(t, f.FeatureID)
		require.NotEmpty(t, f.LayerBodID)
		require.NotEmpty(t, f.LayerName)
		require.NotNil(t, f.Geometry, "geometry should be present")
		require.NotEmpty(t, f.BBox, "bbox should be present")
		require.NotEmpty(t, f.Properties.TitleEn)
		require.NotEmpty(t, f.Properties.SperrungenType)
		require.NotEmpty(t, f.Properties.Land)
		require.NotEmpty(t, f.Properties.Label)
	}
}
