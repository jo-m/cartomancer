-- name: GetLatestStreetlightCreatedAt :one
-- Returns the created_at of the most recent streetlight inserted by a given job.
SELECT created_at FROM streetlights
WHERE inserted_by = ?
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteStreetlightsByInsertedBy :execrows
-- Removes all streetlights inserted by a specific job, so they can be replaced.
DELETE FROM streetlights WHERE inserted_by = ?;

-- name: UpsertStreetlightAttribution :one
-- Returns the id of the attribution with the given credit and URL, inserting it first if needed.
INSERT INTO streetlight_attributions (attribution, attribution_href)
VALUES (?, ?)
ON CONFLICT(attribution, attribution_href) DO UPDATE SET attribution = excluded.attribution
RETURNING id;

-- name: InsertStreetlight :exec
INSERT INTO streetlights (
    uuid, source_id, inserted_by, created_at,
    attribution_id, cell, geometry
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?
);

-- name: CountStreetlights :one
SELECT COUNT(*) FROM streetlights;
