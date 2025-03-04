-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,

    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,

    secret_hash BLOB NOT NULL,
    user_id TEXT,
    data TEXT,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE SET NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
-- +goose StatementEnd
