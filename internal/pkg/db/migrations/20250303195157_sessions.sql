-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,

    created_at DATETIME NOT NULL,
    last_active_at DATETIME NOT NULL,

    user_id TEXT,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE TABLE sessions_data (
    session_id TEXT NOT NULL,
    key TEXT NOT NULL,
    data TEXT NOT NULL,
    FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    UNIQUE (session_id, key)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
DROP TABLE sessions_data;
-- +goose StatementEnd
