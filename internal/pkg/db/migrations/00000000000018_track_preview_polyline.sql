-- +goose Up
ALTER TABLE tracks ADD COLUMN preview_polyline TEXT;

-- +goose Down
ALTER TABLE tracks DROP COLUMN preview_polyline;
