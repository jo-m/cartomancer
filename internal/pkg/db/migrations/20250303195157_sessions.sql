-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
    -- TODO: make this a UUID.
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    secret_hash BLOB NOT NULL,
    data TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
-- +goose StatementEnd
