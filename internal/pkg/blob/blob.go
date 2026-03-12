// Package blob provides helpers for storing and retrieving compressed blobs.
package blob

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	ID      int64
	Content []byte
}

// Create inserts a blob, optionally compressing its content.
// The ID is assigned by the database and returned in the result.
func Create(ctx context.Context, q *db.Queries, content []byte, compression Compression) (Blob, error) {
	hash := sha256.Sum256(content)

	stored := content
	if compression == CompressionZstd {
		var err error
		stored, err = zstd.Compress(nil, content)
		if err != nil {
			return Blob{}, fmt.Errorf("zstd compress: %w", err)
		}
	}

	raw, err := q.CreateBlob(ctx, db.CreateBlobParams{
		Compression: int64(compression),
		Content:     stored,
		HashType:    int64(HashTypeSHA256),
		Hash:        hash[:],
	})
	if err != nil {
		return Blob{}, err
	}

	return Blob{ID: raw.ID, Content: content}, nil
}

// acceptsZstd reports whether the request's Accept-Encoding header includes zstd
// with a non-zero q-value.
func acceptsZstd(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		part = strings.TrimSpace(part)
		enc := part
		q := 1.0
		if i := strings.IndexByte(part, ';'); i >= 0 {
			enc = strings.TrimSpace(part[:i])
			param := strings.TrimSpace(part[i+1:])
			if strings.HasPrefix(param, "q=") {
				if v, err := strconv.ParseFloat(param[2:], 64); err == nil {
					q = v
				}
			}
		}
		if strings.EqualFold(enc, "zstd") && q > 0 {
			return true
		}
	}
	return false
}

// Serve writes the blob identified by id to w, setting the given contentType and filename.
// If the client advertises zstd support in Accept-Encoding and the blob is stored
// zstd-compressed, the raw compressed bytes are forwarded directly with a
// Content-Encoding: zstd header to avoid an unnecessary decompress/recompress cycle.
// An error is returned if the database fetch or decompression fails; note that if the
// error occurs after headers have already been written, the caller cannot send an error
// response.
func Serve(w http.ResponseWriter, r *http.Request, q *db.Queries, id int64, contentType, filename string) error {
	raw, err := q.GetBlob(r.Context(), id)
	if err != nil {
		return err
	}

	comp := Compression(raw.Compression)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))

	if comp == CompressionZstd && acceptsZstd(r) {
		w.Header().Set("Content-Encoding", "zstd")
		w.Header().Set("Content-Length", strconv.Itoa(len(raw.Content)))
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(raw.Content)
		return err
	}

	content := raw.Content
	switch comp {
	case CompressionNone:
		// No decompression needed.
	case CompressionZstd:
		content, err = zstd.Decompress(nil, raw.Content)
		if err != nil {
			return fmt.Errorf("zstd decompress: %w", err)
		}
	default:
		return fmt.Errorf("unknown compression type: %d", raw.Compression)
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(content)
	return err
}

// Get retrieves a blob and decompresses its content if needed.
func Get(ctx context.Context, q *db.Queries, id int64) (Blob, error) {
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
		ID:      raw.ID,
		Content: content,
	}, nil
}
