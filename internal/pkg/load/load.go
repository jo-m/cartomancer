// Package load loads tracks from different file formats.
package load

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// ErrUnsupportedFileExtension is returned when the file extension is not recognized.
var (
	ErrUnsupportedFileExtension = errors.New("unsupported file extension")
)

// Fn is a function that parses a file by name and content into a TrackSource.
type Fn func(filename string, contents io.Reader) (track.TrackSource, error)

var loaders = map[string]Fn{
	".fit": loadFit,
	".gpx": loadGpx,
}

// Blob parses a track from an in-memory blob by inferring the format from filename's extension.
// Returns [ErrUnsupportedFileExtension] if the extension is not supported.
func Blob(filename string, contents io.Reader) (track.TrackSource, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	loader, ok := loaders[ext]
	if !ok {
		return nil, ErrUnsupportedFileExtension
	}

	return loader(filename, contents)
}

// Path loads a track from the file at the given path, inferring the format from the extension.
// Returns [ErrUnsupportedFileExtension] if the extension is not supported.
func Path(path string) (track.TrackSource, error) {
	filename := filepath.Base(path)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open '%s': %w", path, err)
	}
	defer f.Close()

	return Blob(filename, f)
}
