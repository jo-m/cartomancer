package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestSPAHandlerCacheControl(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":             &fstest.MapFile{Data: []byte("<!doctype html><div id=root></div>")},
		"favicon.svg":            &fstest.MapFile{Data: []byte("<svg/>")},
		"assets/index-abc123.js": &fstest.MapFile{Data: []byte("console.log(1)")},
	}

	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantCache    string
		wantContains string
	}{
		{
			name:         "root serves index",
			path:         "/",
			wantStatus:   http.StatusOK,
			wantCache:    cacheControlNoCache,
			wantContains: "<!doctype html>",
		},
		{
			name:         "client route falls back to index",
			path:         "/tracks/abc",
			wantStatus:   http.StatusOK,
			wantCache:    cacheControlNoCache,
			wantContains: "<!doctype html>",
		},
		{
			name:         "unhashed static file is revalidated",
			path:         "/favicon.svg",
			wantStatus:   http.StatusOK,
			wantCache:    cacheControlNoCache,
			wantContains: "<svg/>",
		},
		{
			name:         "hashed asset is immutable",
			path:         "/assets/index-abc123.js",
			wantStatus:   http.StatusOK,
			wantCache:    cacheControlImmutable,
			wantContains: "console.log(1)",
		},
		{
			name:       "missing asset is a 404",
			path:       "/assets/index-gone.js",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			spaHandler(fsys).ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantCache, rec.Header().Get("Cache-Control"))
			if tt.wantContains != "" {
				require.Contains(t, rec.Body.String(), tt.wantContains)
			}
		})
	}
}
