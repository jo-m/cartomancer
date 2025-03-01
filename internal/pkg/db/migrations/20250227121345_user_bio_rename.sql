-- +goose Up
-- +goose StatementBegin
ALTER TABLE users RENAME COLUMN bio TO biography;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users RENAME COLUMN biography TO bio;
-- +goose StatementEnd
