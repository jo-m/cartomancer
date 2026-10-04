//go:build online

package ktzh_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/streetlights/ktzh"
)

func TestOnlineFetch(t *testing.T) {
	ctx := context.Background()

	features, err := ktzh.Fetch(ctx)
	require.NoError(t, err)
	require.Greater(t, len(features), 1000, "expected a substantial number of lamps")

	// Switzerland's WGS84 bounding box (loose).
	const minLat, maxLat = 45.7, 47.9
	const minLon, maxLon = 5.9, 10.6

	withGeom := 0
	for _, f := range features {
		require.True(t, strings.HasPrefix(f.SourceID, "ktzh-"), "source id %q", f.SourceID)
		require.NotEmpty(t, f.Properties)
		if f.Geometry == nil {
			continue
		}
		withGeom++
		// Sample one coordinate of the decoded geometry and make sure it
		// looks like WGS84 inside Switzerland (catches a forgotten axis swap).
		g := f.Geometry.Geometry()
		require.NotNil(t, g, "feature %s has nil orb geometry", f.SourceID)
		min := g.Bound().Min
		require.GreaterOrEqual(t, min.Lat(), minLat, "feature %s lat out of range: %v", f.SourceID, min)
		require.LessOrEqual(t, min.Lat(), maxLat, "feature %s lat out of range: %v", f.SourceID, min)
		require.GreaterOrEqual(t, min.Lon(), minLon, "feature %s lon out of range: %v", f.SourceID, min)
		require.LessOrEqual(t, min.Lon(), maxLon, "feature %s lon out of range: %v", f.SourceID, min)
	}
	require.NotZero(t, withGeom, "expected at least one feature with geometry")
}
