-- name: InsertMapBuild :exec
INSERT INTO map_builds (uuid, created_at, key, size, md5sum, uploaded, version, maxzoom, bbox_min_lon, bbox_min_lat, bbox_max_lon, bbox_max_lat, ready)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: SetMapBuildReady :execresult
UPDATE map_builds SET ready = 1 WHERE uuid = ?;

-- name: GetLatestReadyMapBuild :one
SELECT * FROM map_builds WHERE ready = 1 ORDER BY uploaded DESC LIMIT 1;

-- name: GetLatestMapBuild :one
SELECT * FROM map_builds ORDER BY uploaded DESC LIMIT 1;

-- name: GetMapBuildByKey :one
SELECT * FROM map_builds
WHERE key = ?
  AND maxzoom = ?
  AND bbox_min_lon IS ?
  AND bbox_min_lat IS ?
  AND bbox_max_lon IS ?
  AND bbox_max_lat IS ?
LIMIT 1;

-- name: DeleteMapBuild :execresult
DELETE FROM map_builds WHERE uuid = ?;

-- name: ListMapBuilds :many
SELECT * FROM map_builds ORDER BY uploaded DESC;
