-- +goose Up
CREATE TABLE tags (
    id INTEGER PRIMARY KEY,
    tag TEXT NOT NULL CHECK(LENGTH(tag) > 0),
    user_id TEXT NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    UNIQUE(tag, user_id)
);

CREATE TABLE track_tags (
    track_id TEXT NOT NULL REFERENCES tracks(uuid) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (track_id, tag_id)
);

-- +goose Down
DROP TABLE track_tags;
DROP TABLE tags;
