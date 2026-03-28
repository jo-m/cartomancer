-- name: CreateSegmentJunction :exec
INSERT INTO segment_junctions (uuid, h3_cell, lat, lon, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: CreateSegment :exec
INSERT INTO segments (uuid, start_junction_id, end_junction_id, h3_resolution, distance_m, ascent_m, n_tracks, polyline, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateSegmentTrack :exec
INSERT INTO segment_tracks (segment_id, track_id)
VALUES (?, ?);

-- name: DeleteAllSegments :exec
DELETE FROM segments;

-- name: DeleteAllSegmentJunctions :exec
DELETE FROM segment_junctions;

-- name: DeleteAllSegmentTracks :exec
DELETE FROM segment_tracks;

-- name: ListAllSegments :many
SELECT s.uuid, s.distance_m, s.ascent_m, s.n_tracks, s.h3_resolution, s.polyline,
       sj_start.lat AS start_lat, sj_start.lon AS start_lon,
       sj_end.lat AS end_lat, sj_end.lon AS end_lon
FROM segments s
JOIN segment_junctions sj_start ON sj_start.uuid = s.start_junction_id
JOIN segment_junctions sj_end ON sj_end.uuid = s.end_junction_id
ORDER BY s.distance_m DESC, s.n_tracks DESC;

-- name: ListAllSegmentJunctions :many
SELECT uuid, h3_cell, lat, lon FROM segment_junctions ORDER BY uuid;

-- name: ListSegments :many
SELECT s.uuid, s.distance_m, s.ascent_m, s.n_tracks, s.h3_resolution,
       sj_start.lat AS start_lat, sj_start.lon AS start_lon,
       sj_end.lat AS end_lat, sj_end.lon AS end_lon
FROM segments s
JOIN segment_junctions sj_start ON sj_start.uuid = s.start_junction_id
JOIN segment_junctions sj_end ON sj_end.uuid = s.end_junction_id
ORDER BY s.distance_m DESC, s.n_tracks DESC
LIMIT ? OFFSET ?;

-- name: CountSegments :one
SELECT COUNT(*) FROM segments;

-- name: GetSegment :one
SELECT s.uuid, s.distance_m, s.ascent_m, s.n_tracks, s.h3_resolution, s.polyline,
       s.start_junction_id, s.end_junction_id,
       sj_start.lat AS start_lat, sj_start.lon AS start_lon, sj_start.h3_cell AS start_h3_cell,
       sj_end.lat AS end_lat, sj_end.lon AS end_lon, sj_end.h3_cell AS end_h3_cell
FROM segments s
JOIN segment_junctions sj_start ON sj_start.uuid = s.start_junction_id
JOIN segment_junctions sj_end ON sj_end.uuid = s.end_junction_id
WHERE s.uuid = ?;

-- name: ListSegmentTrackUUIDs :many
SELECT st.track_id FROM segment_tracks st WHERE st.segment_id = ?;

-- name: ListSegmentsByTrack :many
SELECT s.uuid, s.distance_m, s.ascent_m, s.n_tracks, s.h3_resolution,
       sj_start.lat AS start_lat, sj_start.lon AS start_lon,
       sj_end.lat AS end_lat, sj_end.lon AS end_lon
FROM segment_tracks st
JOIN segments s ON s.uuid = st.segment_id
JOIN segment_junctions sj_start ON sj_start.uuid = s.start_junction_id
JOIN segment_junctions sj_end ON sj_end.uuid = s.end_junction_id
WHERE st.track_id = ?
ORDER BY s.distance_m DESC, s.n_tracks DESC;

-- name: ListAllGroupableTracks :many
SELECT uuid, blob_id, file_format, original_filename, total_distance_m, track_type
FROM tracks
WHERE total_distance_m <= ?
ORDER BY uuid;
