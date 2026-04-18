-- +goose Up
CREATE TABLE track_comments (
    uuid TEXT PRIMARY KEY,
    track_id TEXT NOT NULL REFERENCES tracks(uuid) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    body TEXT NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX idx_track_comments_track_id ON track_comments(track_id);

-- +goose Down
DROP INDEX idx_track_comments_track_id;
DROP TABLE track_comments;
