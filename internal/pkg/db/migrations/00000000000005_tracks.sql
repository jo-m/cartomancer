-- +goose Up
-- +goose StatementBegin
CREATE TABLE tracks (
    id TEXT PRIMARY KEY,

    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,

    user_id TEXT NOT NULL,

    blob_id TEXT NOT NULL,
    file_format INTEGER NOT NULL,

    name TEXT NOT NULL,
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

    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(blob_id) REFERENCES blobs(id) ON DELETE RESTRICT
);
-- When a track is deleted, delete its blob (unless another track still references it).
CREATE TRIGGER tracks_delete_blob AFTER DELETE ON tracks
BEGIN
    DELETE FROM blobs WHERE id = OLD.blob_id
    AND NOT EXISTS (SELECT 1 FROM tracks WHERE blob_id = OLD.blob_id);
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER tracks_delete_blob;
DROP TABLE tracks;
-- +goose StatementEnd
