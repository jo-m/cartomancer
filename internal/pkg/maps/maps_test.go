package maps

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testBbox is a bounding box covering approximately Switzerland, used in tests.
const testBbox = "5.5,45.5,11.0,48.2"

func TestLatestBuild_empty(t *testing.T) {
	_, err := LatestBuild(nil)
	require.Error(t, err)
}

func TestLatestBuild_single(t *testing.T) {
	b := BuildMetadata{Key: "20260101.pmtiles", Uploaded: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	got, err := LatestBuild([]BuildMetadata{b})
	require.NoError(t, err)
	require.Equal(t, b.Key, got.Key)
}

func TestLatestBuild_multiple(t *testing.T) {
	builds := []BuildMetadata{
		{Key: "20260101.pmtiles", Uploaded: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Key: "20260315.pmtiles", Uploaded: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)},
		{Key: "20260210.pmtiles", Uploaded: time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)},
	}
	got, err := LatestBuild(builds)
	require.NoError(t, err)
	require.Equal(t, "20260315.pmtiles", got.Key)
}

func TestOutputPath(t *testing.T) {
	got := OutputPath("/data/maps", "abc-123")
	require.Equal(t, "/data/maps/abc-123.pmtiles", got)
}
