// Package load loads tracks from different file formats.
package load

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"jo-m.ch/go/detour/internal/pkg/track"
)

var (
	ErrUnsupportedFileExtension = errors.New("unsupported file extension")
)

type LoadFn func(filename string, contents io.Reader) (track.TrackSource, error)

var loaders map[string]LoadFn = map[string]LoadFn{
	".fit": loadFit,
	".gpx": loadGpx,
}

func Blob(filename string, contents io.Reader) (track.TrackSource, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	loader, ok := loaders[ext]
	if !ok {
		return nil, ErrUnsupportedFileExtension
	}

	return loader(filename, contents)
}

func Path(path string) (track.TrackSource, error) {
	filename := filepath.Base(path)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open '%s': %w", path, err)
	}
	defer f.Close()

	return Blob(filename, f)
}
