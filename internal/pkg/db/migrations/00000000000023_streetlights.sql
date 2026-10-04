-- +goose Up
-- +goose StatementBegin
CREATE TABLE streetlight_attributions (
    id INTEGER PRIMARY KEY,

    -- attribution is the human-readable data source credit.
    attribution TEXT NOT NULL,

    -- attribution_href is the URL for the data source.
    attribution_href TEXT NOT NULL,

    UNIQUE(attribution, attribution_href)
);

CREATE TABLE streetlights (
    uuid TEXT PRIMARY KEY,

    -- source_id is the identifier from the upstream data source (e.g. feature
    -- id or object id). Not necessarily numeric; format depends on the source.
    source_id TEXT NOT NULL,

    -- inserted_by identifies which job inserted this row, so that each job can
    -- delete only its own rows during a refresh cycle.
    inserted_by TEXT NOT NULL,

    created_at DATETIME NOT NULL,

    -- attribution_id references the data source credit.
    attribution_id INTEGER NOT NULL REFERENCES streetlight_attributions(id),

    -- cell is the H3 cell index at resolution 9 covering the lamp position.
    cell INTEGER NOT NULL,

    -- geometry is the GeoJSON Point in WGS84, stored as a JSON text string.
    geometry TEXT NOT NULL
);

CREATE INDEX idx_streetlights_inserted_by ON streetlights (inserted_by);

CREATE INDEX idx_streetlights_cell ON streetlights (cell);

-- created_at is indexed for the cache-validator fingerprint query, which
-- reads count and max(created_at) on every coverage request.
CREATE INDEX idx_streetlights_created_at ON streetlights (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS streetlights;
DROP TABLE IF EXISTS streetlight_attributions;
-- +goose StatementEnd
