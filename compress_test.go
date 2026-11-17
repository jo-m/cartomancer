package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"
)

func TestCompressExceptRanges(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/a.js": &fstest.MapFile{Data: []byte("console.log(1234567890);")},
	}
	h := compressExceptRanges(middleware.Compress(5))(spaHandler(fsys))

	t.Run("full response is compressed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/assets/a.js", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
		require.Equal(t, cacheControlImmutable, rec.Header().Get("Cache-Control"))
	})

	t.Run("range response is not compressed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/assets/a.js", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Range", "bytes=0-9")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		require.Equal(t, http.StatusPartialContent, rec.Code)
		require.Empty(t, rec.Header().Get("Content-Encoding"))
		require.Equal(t, "bytes 0-9/24", rec.Header().Get("Content-Range"))
		require.Equal(t, "console.lo", rec.Body.String())
	})
}
