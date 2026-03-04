-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    uuid TEXT PRIMARY KEY,

    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    -- Updating those does not update the `updated_at` field.
    last_login_at DATETIME,
    last_active_at DATETIME,

    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    otp_secret BLOB DEFAULT NULL,
    admin INTEGER NOT NULL DEFAULT 0
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
