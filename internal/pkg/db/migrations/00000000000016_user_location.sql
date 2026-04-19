-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN location_name TEXT;
ALTER TABLE users ADD COLUMN location_lat REAL;
ALTER TABLE users ADD COLUMN location_lon REAL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN location_name;
ALTER TABLE users DROP COLUMN location_lat;
ALTER TABLE users DROP COLUMN location_lon;
-- +goose StatementEnd
