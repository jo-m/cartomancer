-- +goose Up
-- +goose StatementBegin
CREATE TABLE blobs (
    id TEXT PRIMARY KEY,

    filename TEXT NOT NULL,
    compression INTEGER NOT NULL,
    content BLOB NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE blobs;
-- +goose StatementEnd
