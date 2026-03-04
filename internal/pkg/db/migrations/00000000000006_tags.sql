-- +goose Up
CREATE TABLE tags (
    id INTEGER PRIMARY KEY,
    tag TEXT NOT NULL UNIQUE
);

CREATE TABLE track_tags (
    track_id TEXT NOT NULL REFERENCES tracks(uuid) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (track_id, tag_id)
);

-- +goose Down
DROP TABLE track_tags;
DROP TABLE tags;
