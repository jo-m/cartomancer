-- +goose NO TRANSACTION
-- The preview polyline columns are populated on upload and the backfill job
-- has run for every existing row in production, so they can be promoted to
-- NOT NULL. SQLite cannot ALTER a column to add NOT NULL, so the standard
-- table-rebuild procedure is used (https://sqlite.org/lang_altertable.html).

-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN TRANSACTION;

-- +goose StatementBegin
CREATE TABLE tracks_new (
    uuid TEXT PRIMARY KEY,

    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,

    initial_editing_completed INTEGER NOT NULL DEFAULT 0 CHECK(initial_editing_completed IN (0, 1)),

    user_id TEXT NOT NULL,
    public INTEGER NOT NULL DEFAULT 0 CHECK(public IN (0, 1)),

    blob_id INTEGER NOT NULL,
    file_format INTEGER NOT NULL,
    original_filename TEXT NOT NULL CHECK(LENGTH(original_filename) > 0),

    name TEXT NOT NULL CHECK(LENGTH(name) > 0),
    description TEXT,
    source TEXT,
    author TEXT,
    author_link_url TEXT,

    track_type INTEGER NOT NULL,
    link_url TEXT,

    sport INTEGER NOT NULL,
    sub_sport INTEGER NOT NULL,

    total_distance_m REAL NOT NULL,
    total_ascent_m REAL NOT NULL,

    start_lat REAL,
    start_lon REAL,
    end_lat REAL,
    end_lon REAL,

    bounds_min_lat REAL,
    bounds_min_lon REAL,
    bounds_max_lat REAL,
    bounds_max_lon REAL,

    min_elevation_m REAL,
    max_elevation_m REAL,

    original_created_at DATETIME,

    polyline_dp5m_varint BLOB NOT NULL,
    polyline_dp50m_varint BLOB NOT NULL,

    FOREIGN KEY(user_id) REFERENCES users(uuid) ON DELETE CASCADE,
    FOREIGN KEY(blob_id) REFERENCES blobs(id) ON DELETE RESTRICT
);
-- +goose StatementEnd

INSERT INTO tracks_new (
    uuid, created_at, updated_at, initial_editing_completed, user_id, public,
    blob_id, file_format, original_filename,
    name, description, source, author, author_link_url,
    track_type, link_url, sport, sub_sport,
    total_distance_m, total_ascent_m,
    start_lat, start_lon, end_lat, end_lon,
    bounds_min_lat, bounds_min_lon, bounds_max_lat, bounds_max_lon,
    min_elevation_m, max_elevation_m,
    original_created_at,
    polyline_dp5m_varint, polyline_dp50m_varint
) SELECT
    uuid, created_at, updated_at, initial_editing_completed, user_id, public,
    blob_id, file_format, original_filename,
    name, description, source, author, author_link_url,
    track_type, link_url, sport, sub_sport,
    total_distance_m, total_ascent_m,
    start_lat, start_lon, end_lat, end_lon,
    bounds_min_lat, bounds_min_lon, bounds_max_lat, bounds_max_lon,
    min_elevation_m, max_elevation_m,
    original_created_at,
    polyline_dp5m_varint, polyline_dp50m_varint
FROM tracks;

DROP TABLE tracks;

ALTER TABLE tracks_new RENAME TO tracks;

-- +goose StatementBegin
CREATE TRIGGER tracks_delete_blob AFTER DELETE ON tracks
BEGIN
    DELETE FROM blobs WHERE id = OLD.blob_id
    AND NOT EXISTS (SELECT 1 FROM tracks WHERE blob_id = OLD.blob_id);
END;
-- +goose StatementEnd

PRAGMA foreign_key_check;

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

BEGIN TRANSACTION;

-- +goose StatementBegin
CREATE TABLE tracks_new (
    uuid TEXT PRIMARY KEY,

    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,

    initial_editing_completed INTEGER NOT NULL DEFAULT 0 CHECK(initial_editing_completed IN (0, 1)),

    user_id TEXT NOT NULL,
    public INTEGER NOT NULL DEFAULT 0 CHECK(public IN (0, 1)),

    blob_id INTEGER NOT NULL,
    file_format INTEGER NOT NULL,
    original_filename TEXT NOT NULL CHECK(LENGTH(original_filename) > 0),

    name TEXT NOT NULL CHECK(LENGTH(name) > 0),
    description TEXT,
    source TEXT,
    author TEXT,
    author_link_url TEXT,

    track_type INTEGER NOT NULL,
    link_url TEXT,

    sport INTEGER NOT NULL,
    sub_sport INTEGER NOT NULL,

    total_distance_m REAL NOT NULL,
    total_ascent_m REAL NOT NULL,

    start_lat REAL,
    start_lon REAL,
    end_lat REAL,
    end_lon REAL,

    bounds_min_lat REAL,
    bounds_min_lon REAL,
    bounds_max_lat REAL,
    bounds_max_lon REAL,

    min_elevation_m REAL,
    max_elevation_m REAL,

    original_created_at DATETIME,

    polyline_dp5m_varint BLOB,
    polyline_dp50m_varint BLOB,

    FOREIGN KEY(user_id) REFERENCES users(uuid) ON DELETE CASCADE,
    FOREIGN KEY(blob_id) REFERENCES blobs(id) ON DELETE RESTRICT
);
-- +goose StatementEnd

INSERT INTO tracks_new (
    uuid, created_at, updated_at, initial_editing_completed, user_id, public,
    blob_id, file_format, original_filename,
    name, description, source, author, author_link_url,
    track_type, link_url, sport, sub_sport,
    total_distance_m, total_ascent_m,
    start_lat, start_lon, end_lat, end_lon,
    bounds_min_lat, bounds_min_lon, bounds_max_lat, bounds_max_lon,
    min_elevation_m, max_elevation_m,
    original_created_at,
    polyline_dp5m_varint, polyline_dp50m_varint
) SELECT
    uuid, created_at, updated_at, initial_editing_completed, user_id, public,
    blob_id, file_format, original_filename,
    name, description, source, author, author_link_url,
    track_type, link_url, sport, sub_sport,
    total_distance_m, total_ascent_m,
    start_lat, start_lon, end_lat, end_lon,
    bounds_min_lat, bounds_min_lon, bounds_max_lat, bounds_max_lon,
    min_elevation_m, max_elevation_m,
    original_created_at,
    polyline_dp5m_varint, polyline_dp50m_varint
FROM tracks;

DROP TABLE tracks;

ALTER TABLE tracks_new RENAME TO tracks;

-- +goose StatementBegin
CREATE TRIGGER tracks_delete_blob AFTER DELETE ON tracks
BEGIN
    DELETE FROM blobs WHERE id = OLD.blob_id
    AND NOT EXISTS (SELECT 1 FROM tracks WHERE blob_id = OLD.blob_id);
END;
-- +goose StatementEnd

PRAGMA foreign_key_check;

COMMIT;

PRAGMA foreign_keys = ON;
