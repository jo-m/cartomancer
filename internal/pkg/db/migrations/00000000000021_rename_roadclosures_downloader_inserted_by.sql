-- +goose Up
-- +goose StatementBegin
-- Rename rows previously written by the monolithic roadclosures downloader
-- so they line up with the new per-source job kind, allowing the astra
-- downloader to keep refreshing them via DeleteRoadClosuresByInsertedBy.
UPDATE road_closures
SET inserted_by = 'roadclosures.astra.downloader'
WHERE inserted_by = 'roadclosures.downloader';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE road_closures
SET inserted_by = 'roadclosures.downloader'
WHERE inserted_by = 'roadclosures.astra.downloader';
-- +goose StatementEnd
