package blob

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/DataDog/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/db"
)

func TestCreateUncompressed(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := t.Context()
	q := d.QueryRW()

	content := []byte("hello world")
	blob, err := Create(ctx, q, "id1", content, CompressionNone)
	require.NoError(t, err)

	assert.Equal(t, "id1", blob.ID)
	assert.Equal(t, content, blob.Content)

	// Verify raw DB row is uncompressed.
	raw, err := q.GetBlob(ctx, "id1")
	require.NoError(t, err)
	assert.Equal(t, int64(CompressionNone), raw.Compression)
	assert.Equal(t, content, raw.Content)
}

func TestCreateCompressed(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := t.Context()
	q := d.QueryRW()

	content := []byte("hello world, this is some content that should be compressed")
	blob, err := Create(ctx, q, "id1", content, CompressionZstd)
	require.NoError(t, err)

	assert.Equal(t, "id1", blob.ID)
	assert.Equal(t, content, blob.Content)

	// Verify raw DB row is compressed (content differs).
	raw, err := q.GetBlob(ctx, "id1")
	require.NoError(t, err)
	assert.Equal(t, int64(CompressionZstd), raw.Compression)
	assert.NotEqual(t, content, raw.Content)
}

func TestGetUncompressed(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := t.Context()
	q := d.QueryRW()

	content := []byte("raw content")
	_, err := Create(ctx, q, "id1", content, CompressionNone)
	require.NoError(t, err)

	blob, err := Get(ctx, q, "id1")
	require.NoError(t, err)
	assert.Equal(t, "id1", blob.ID)
	assert.Equal(t, content, blob.Content)
}

func TestGetCompressed(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := t.Context()
	q := d.QueryRW()

	content := []byte("compressed content that should round-trip correctly")
	_, err := Create(ctx, q, "id1", content, CompressionZstd)
	require.NoError(t, err)

	blob, err := Get(ctx, q, "id1")
	require.NoError(t, err)
	assert.Equal(t, "id1", blob.ID)
	assert.Equal(t, content, blob.Content)
}

func TestGetNotFound(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := t.Context()
	q := d.QueryRW()

	_, err := Get(ctx, q, "nonexistent")
	assert.Error(t, err)
}

func TestServeUncompressed(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	q := d.QueryRW()

	content := []byte("hello world")
	_, err := Create(t.Context(), q, "id1", content, CompressionNone)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	require.NoError(t, Serve(w, r, q, "id1", "text/plain", "file.txt"))

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "file.txt")
	assert.Empty(t, resp.Header.Get("Content-Encoding"))
	assert.Equal(t, strconv.Itoa(len(content)), resp.Header.Get("Content-Length"))
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, content, body)
}

func TestServeCompressedNoZstd(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	q := d.QueryRW()

	content := []byte("content for clients without zstd support")
	_, err := Create(t.Context(), q, "id1", content, CompressionZstd)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	require.NoError(t, Serve(w, r, q, "id1", "text/plain", "file.txt"))

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Content-Encoding"))
	assert.Equal(t, strconv.Itoa(len(content)), resp.Header.Get("Content-Length"))
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, content, body)
}

func TestServeCompressedWithZstd(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	q := d.QueryRW()

	content := []byte("content for zstd-capable clients, served as raw compressed bytes")
	_, err := Create(t.Context(), q, "id1", content, CompressionZstd)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Encoding", "zstd, gzip")
	w := httptest.NewRecorder()
	require.NoError(t, Serve(w, r, q, "id1", "text/plain", "file.txt"))

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "zstd", resp.Header.Get("Content-Encoding"))
	// Body is raw zstd bytes; decompressing must recover the original content.
	body, _ := io.ReadAll(resp.Body)
	decompressed, err := zstd.Decompress(nil, body)
	require.NoError(t, err)
	assert.Equal(t, content, decompressed)
}

func TestServeCompressedZstdQZero(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	q := d.QueryRW()

	content := []byte("content for client that explicitly rejects zstd")
	_, err := Create(t.Context(), q, "id1", content, CompressionZstd)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Encoding", "zstd;q=0, gzip")
	w := httptest.NewRecorder()
	require.NoError(t, Serve(w, r, q, "id1", "text/plain", "file.txt"))

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Content-Encoding"))
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, content, body)
}

func TestServeNotFound(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	q := d.QueryRW()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	assert.Error(t, Serve(w, r, q, "nonexistent", "text/plain", "file.txt"))
}
