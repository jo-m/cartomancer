-- name: CreateBlob :one
INSERT INTO blobs (
  uuid, filename, compression, content, hash_type, hash
) VALUES (
  ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetBlob :one
SELECT * FROM blobs
WHERE uuid = ? LIMIT 1;
