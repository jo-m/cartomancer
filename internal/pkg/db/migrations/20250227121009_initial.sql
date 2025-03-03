-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    -- TODO: make this a UUID.
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    last_seen_at DATETIME, -- TODO: fill, or use last_login...

    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
