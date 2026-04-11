//go:build online

package maps_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/maps"
)

// TestOnlineFetchBuilds downloads the protomaps builds.json index and verifies
// that it contains at least one entry with plausible metadata.
func TestOnlineFetchBuilds(t *testing.T) {
	ctx := context.Background()

	builds, err := maps.FetchBuilds(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, builds, "expected at least one build entry")

	for _, b := range builds {
		require.NotEmpty(t, b.Key, "key must not be empty")
		require.Contains(t, b.Key, ".pmtiles", "key should end with .pmtiles")
		require.Greater(t, b.Size, int64(0), "size must be positive")
		require.False(t, b.Uploaded.IsZero(), "uploaded must not be zero")
		require.NotEmpty(t, b.Version, "version must not be empty")
	}

	// The latest build should always have checksums.
	latest, err := maps.LatestBuild(builds)
	require.NoError(t, err)
	require.NotEmpty(t, latest.MD5Sum, "latest build should have md5sum")
	require.NotEmpty(t, latest.B3Sum, "latest build should have b3sum")
}

// TestOnlineLatestBuild verifies that [maps.LatestBuild] returns the most recent
// entry from the live builds.json index.
func TestOnlineLatestBuild(t *testing.T) {
	ctx := context.Background()

	builds, err := maps.FetchBuilds(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, builds)

	latest, err := maps.LatestBuild(builds)
	require.NoError(t, err)
	require.NotEmpty(t, latest.Key)

	// The latest build must not be older than any other.
	for _, b := range builds {
		require.False(t, b.Uploaded.After(latest.Uploaded),
			"build %s (uploaded %s) is newer than latest %s (uploaded %s)",
			b.Key, b.Uploaded, latest.Key, latest.Uploaded)
	}
}
