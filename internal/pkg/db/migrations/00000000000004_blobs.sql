-- +goose Up
-- +goose StatementBegin
CREATE TABLE blobs (
    id INTEGER PRIMARY KEY,

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
