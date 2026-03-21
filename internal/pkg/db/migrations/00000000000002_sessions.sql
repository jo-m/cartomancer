-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
    uuid TEXT PRIMARY KEY,

    created_at DATETIME NOT NULL,
    last_active_at DATETIME NOT NULL,

    user_id TEXT,
    FOREIGN KEY(user_id) REFERENCES users(uuid) ON DELETE CASCADE
) WITHOUT ROWID;
CREATE TABLE sessions_data (
    session_id TEXT NOT NULL CHECK(LENGTH(session_id) > 0),
    key TEXT NOT NULL CHECK(LENGTH(key) > 0),
    data TEXT NOT NULL CHECK(LENGTH(data) > 0),
    FOREIGN KEY(session_id) REFERENCES sessions(uuid) ON DELETE CASCADE,
    PRIMARY KEY (session_id, key)
) WITHOUT ROWID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
DROP TABLE sessions_data;
-- +goose StatementEnd
