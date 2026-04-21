package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

const staticDir = "static"

//go:embed static
var staticEmbed embed.FS

func getStaticFS(devMode bool) (fs.FS, error) {
	if devMode {
		return os.DirFS(staticDir), nil
	}
	return fs.Sub(staticEmbed, staticDir)
}

// spaHandler serves static files and falls back to index.html for unknown paths,
// enabling client-side routing.
func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			f, err := fsys.Open(path)
			if err == nil {
				stat, statErr := f.Stat()
				_ = f.Close()
				if statErr == nil && !stat.IsDir() {
					fileServer.ServeHTTP(w, r)
					return
				}
			}
		}
		http.ServeFileFS(w, r, fsys, "index.html")
	})
}
