-- name: GetLatestRoadClosureCreatedAt :one
-- Returns the created_at of the most recent road closure inserted by a given job.
SELECT created_at FROM road_closures
WHERE inserted_by = ?
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteRoadClosuresByInsertedBy :execrows
-- Removes all road closures inserted by a specific job, so they can be replaced.
DELETE FROM road_closures WHERE inserted_by = ?;

-- name: InsertRoadClosure :exec
INSERT INTO road_closures (
    uuid, source_id, inserted_by, created_at,
    type, starts_at, ends_at, reason,
    title, description, content_provider, geometry
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?
);

-- name: InsertRoadClosureCellRes7 :exec
INSERT OR IGNORE INTO road_closure_cells_res7 (road_closure_id, cell)
VALUES (?, ?);

-- name: CountRoadClosures :one
SELECT COUNT(*) FROM road_closures;

-- name: CountRoadClosureCellsRes7 :one
SELECT COUNT(*) FROM road_closure_cells_res7;
