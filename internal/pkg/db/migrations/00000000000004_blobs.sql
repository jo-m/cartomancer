-- +goose Up
-- +goose StatementBegin
CREATE TABLE blobs (
    uuid TEXT PRIMARY KEY,

    filename TEXT NOT NULL,
    compression INTEGER NOT NULL,
    content BLOB NOT NULL,

    hash_type INTEGER NOT NULL,
    hash BLOB NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE blobs;
-- +goose StatementEnd
