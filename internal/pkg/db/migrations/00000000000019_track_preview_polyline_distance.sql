-- +goose Up
-- The varint preview polyline format was extended to also encode the
-- cumulative distance after the elevation delta. Existing rows are in the
-- old (lat, lon, elevation) format and would decode incorrectly; null them
-- so the backfill job recomputes them in the new format.
-- +goose StatementBegin
UPDATE tracks SET polyline_dp5m_varint = NULL, polyline_dp50m_varint = NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- If we migrate down, we have to do the same.
UPDATE tracks SET polyline_dp5m_varint = NULL, polyline_dp50m_varint = NULL;
-- +goose StatementEnd
