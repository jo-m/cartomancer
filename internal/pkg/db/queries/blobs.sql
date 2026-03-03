-- name: CreateBlob :one
INSERT INTO blobs (
  id, filename, compression, content
) VALUES (
  ?, ?, ?, ?
)
RETURNING *;

-- name: GetBlob :one
SELECT * FROM blobs
WHERE id = ? LIMIT 1;
