-- +goose Up
ALTER TABLE tracks ADD COLUMN polyline_dp5m_varint BLOB;
ALTER TABLE tracks ADD COLUMN polyline_dp50m_varint BLOB;

-- +goose Down
ALTER TABLE tracks DROP COLUMN polyline_dp5m_varint;
ALTER TABLE tracks DROP COLUMN polyline_dp50m_varint;
