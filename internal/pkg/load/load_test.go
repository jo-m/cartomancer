package load

import (
	"path/filepath"
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/track"
)

type snapshot struct {
	Metadata   track.Metadata
	PointCount int
}

func TestGPXSnapshot(t *testing.T) {
	files, err := filepath.Glob("testdata/*.gpx")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := Path(f)
			require.NoError(t, err)

			count := 0
			for range src.All() {
				count++
			}

			golden.Verify(t, snapshot{
				Metadata:   src.Metadata(),
				PointCount: count,
			}, golden.Extension(".json")) // golden.WaitApproval()
		})
	}
}
