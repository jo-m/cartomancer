-- name: GetLatestStreetlightCreatedAt :one
-- Returns the created_at of the most recent streetlight inserted by a given job.
SELECT created_at FROM streetlights
WHERE inserted_by = ?
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteStreetlightsByInsertedBy :execrows
-- Removes all streetlights inserted by a specific job, so they can be replaced.
DELETE FROM streetlights WHERE inserted_by = ?;

-- name: InsertStreetlight :exec
INSERT INTO streetlights (
    uuid, source_id, inserted_by, created_at,
    content_provider, properties, geometry,
    attribution, attribution_href
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?
);

-- name: InsertStreetlightCellRes9 :exec
INSERT OR IGNORE INTO streetlight_cells_res9 (streetlight_id, cell)
VALUES (?, ?);

-- name: CountStreetlights :one
SELECT COUNT(*) FROM streetlights;

-- name: CountStreetlightCellsRes9 :one
SELECT COUNT(*) FROM streetlight_cells_res9;
