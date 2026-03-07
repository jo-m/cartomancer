package blob

import (
	"testing"

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
