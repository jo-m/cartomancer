// Package blob provides helpers for storing and retrieving compressed blobs.
package blob

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/DataDog/zstd"
	"jo-m.ch/go/detour/internal/pkg/db"
)

// Compression represents the compression algorithm used for a blob.
type Compression int

// Compression enum values matching the DB schema.
const (
	CompressionNone Compression = iota
	CompressionZstd
)

// HashType represents the hash algorithm used for a blob.
type HashType int

// HashType enum values matching the DB schema.
const (
	HashTypeSHA256 HashType = iota
)

// Blob is a decompressed blob, ready to use.
type Blob struct {
	ID       string
	Filename string
	Content  []byte
}

// Create inserts a blob, optionally compressing its content.
func Create(ctx context.Context, q *db.Queries, id, filename string, content []byte, compression Compression) (Blob, error) {
	hash := sha256.Sum256(content)

	stored := content
	if compression == CompressionZstd {
		var err error
		stored, err = zstd.Compress(nil, content)
		if err != nil {
			return Blob{}, fmt.Errorf("zstd compress: %w", err)
		}
	}

	_, err := q.CreateBlob(ctx, db.CreateBlobParams{
		Uuid:        id,
		Filename:    filename,
		Compression: int64(compression),
		Content:     stored,
		HashType:    int64(HashTypeSHA256),
		Hash:        hash[:],
	})
	if err != nil {
		return Blob{}, err
	}

	return Blob{ID: id, Filename: filename, Content: content}, nil
}

// Get retrieves a blob and decompresses its content if needed.
func Get(ctx context.Context, q *db.Queries, id string) (Blob, error) {
	raw, err := q.GetBlob(ctx, id)
	if err != nil {
		return Blob{}, err
	}

	content := raw.Content
	switch Compression(raw.Compression) {
	case CompressionNone:
	case CompressionZstd:
		content, err = zstd.Decompress(nil, raw.Content)
		if err != nil {
			return Blob{}, fmt.Errorf("zstd decompress: %w", err)
		}
	default:
		return Blob{}, fmt.Errorf("unknown compression type: %d", raw.Compression)
	}

	return Blob{
		ID:       raw.Uuid,
		Filename: raw.Filename,
		Content:  content,
	}, nil
}
