-- +goose Up
CREATE TABLE track_stars (
    track_id TEXT NOT NULL REFERENCES tracks(uuid) ON DELETE CASCADE,
    user_id  TEXT NOT NULL REFERENCES users(uuid)  ON DELETE CASCADE,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (track_id, user_id)
) WITHOUT ROWID;

-- +goose Down
DROP TABLE track_stars;
