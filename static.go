package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

const staticDir = "static"

// assetsDir holds Vite's content-hashed build output (JS, CSS, fonts).
const assetsDir = "assets/"

const (
	cacheControlImmutable = "public, max-age=31536000, immutable"
	cacheControlNoCache   = "no-cache"
)

//go:embed static
var staticEmbed embed.FS

func getStaticFS(devMode bool) (fs.FS, error) {
	if devMode {
		return os.DirFS(staticDir), nil
	}
	return fs.Sub(staticEmbed, staticDir)
}

// applyCacheControl sets the Cache-Control header for a file served from fsys.
// Files under assetsDir are content-hashed by Vite and can be cached forever.
// Everything else (index.html, favicon.svg) has a stable name, so clients must
// revalidate to pick up new builds.
func applyCacheControl(w http.ResponseWriter, path string) {
	if strings.HasPrefix(path, assetsDir) {
		w.Header().Set("Cache-Control", cacheControlImmutable)
		return
	}
	w.Header().Set("Cache-Control", cacheControlNoCache)
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
					applyCacheControl(w, path)
					fileServer.ServeHTTP(w, r)
					return
				}
			}
		}
		applyCacheControl(w, "index.html")
		http.ServeFileFS(w, r, fsys, "index.html")
	})
}
