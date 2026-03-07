-- +goose Up
-- +goose StatementBegin
CREATE TABLE tracks (
    uuid TEXT PRIMARY KEY,

    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,

    initial_editing_completed INTEGER NOT NULL DEFAULT 0 CHECK(initial_editing_completed IN (0, 1)),

    user_id TEXT NOT NULL,
    public INTEGER NOT NULL DEFAULT 0 CHECK(public IN (0, 1)),

    blob_id TEXT NOT NULL,
    file_format INTEGER NOT NULL,

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

    original_created_at DATETIME,

    FOREIGN KEY(user_id) REFERENCES users(uuid) ON DELETE CASCADE,
    FOREIGN KEY(blob_id) REFERENCES blobs(uuid) ON DELETE RESTRICT
);
-- When a track is deleted, delete its blob (unless another track still references it).
CREATE TRIGGER tracks_delete_blob AFTER DELETE ON tracks
BEGIN
    DELETE FROM blobs WHERE uuid = OLD.blob_id
    AND NOT EXISTS (SELECT 1 FROM tracks WHERE blob_id = OLD.blob_id);
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER tracks_delete_blob;
DROP TABLE tracks;
-- +goose StatementEnd
