-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
    uuid TEXT PRIMARY KEY,

    created_at DATETIME NOT NULL,
    last_active_at DATETIME NOT NULL,

    user_id TEXT,
    FOREIGN KEY(user_id) REFERENCES users(uuid) ON DELETE CASCADE
);
CREATE TABLE sessions_data (
    session_id TEXT NOT NULL,
    key TEXT NOT NULL,
    data TEXT NOT NULL,
    FOREIGN KEY(session_id) REFERENCES sessions(uuid) ON DELETE CASCADE,
    UNIQUE (session_id, key)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
DROP TABLE sessions_data;
-- +goose StatementEnd
