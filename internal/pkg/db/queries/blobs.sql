-- name: CreateBlob :one
INSERT INTO blobs (
  compression, content, hash_type, hash
) VALUES (
  ?, ?, ?, ?
)
RETURNING *;

-- name: GetBlob :one
SELECT * FROM blobs
WHERE id = ? LIMIT 1;

-- name: TrackExistsByUserAndBlobHash :one
SELECT t.uuid FROM tracks t
JOIN blobs b ON b.id = t.blob_id
WHERE t.user_id = ? AND b.hash = ?
LIMIT 1;
