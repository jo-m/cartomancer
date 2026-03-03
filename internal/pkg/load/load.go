// Package load loads tracks from different file formats.
package load

import (
	"log/slog"
	"path/filepath"

	"jo-m.ch/go/detour/internal/pkg/track"
)

type LoadFn func(path string) (track.TrackSource, error)

var loaders map[string]LoadFn = map[string]LoadFn{
	".fit": loadFit,
	".gpx": loadGpx,
}

func One(path string) (track.TrackSource, error) {
	loader, ok := loaders[filepath.Ext(path)]
	if !ok {
		slog.Info("skipping", "path", path)
		return nil, nil
	}

	src, err := loader(path)
	if err != nil {
		return nil, err
	}

	return src, nil
}
