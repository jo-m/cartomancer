-- +goose Up
CREATE TABLE track_groups (
    uuid TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    user_id TEXT NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(uuid) ON DELETE CASCADE
) WITHOUT ROWID;

CREATE TABLE track_group_members (
    group_id TEXT NOT NULL REFERENCES track_groups(uuid) ON DELETE CASCADE,
    track_id TEXT NOT NULL REFERENCES tracks(uuid) ON DELETE CASCADE,
    PRIMARY KEY (group_id, track_id)
) WITHOUT ROWID;

-- +goose Down
DROP TABLE track_group_members;
DROP TABLE track_groups;
