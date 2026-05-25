-- +goose Up
-- +goose StatementBegin
ALTER TABLE track_forecasts
    ADD COLUMN uv_dose_sed REAL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE track_forecasts
    DROP COLUMN uv_dose_sed;
-- +goose StatementEnd
