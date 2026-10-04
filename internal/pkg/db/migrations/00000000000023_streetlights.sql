-- +goose Up
-- +goose StatementBegin
CREATE TABLE streetlights (
    uuid TEXT PRIMARY KEY,

    -- source_id is the identifier from the upstream data source (e.g. feature
    -- id or object id). Not necessarily numeric; format depends on the source.
    source_id TEXT NOT NULL,

    -- inserted_by identifies which job inserted this row, so that each job can
    -- delete only its own rows during a refresh cycle.
    inserted_by TEXT NOT NULL,

    created_at DATETIME NOT NULL,

    -- content_provider names the organization that operates the lights.
    content_provider TEXT,

    -- properties holds the source's feature attributes verbatim as a JSON
    -- object; the shape depends on the source.
    properties TEXT NOT NULL DEFAULT '{}',

    -- geometry is the GeoJSON Point in WGS84, stored as a JSON text string.
    geometry TEXT NOT NULL,

    -- attribution is the human-readable data source credit.
    attribution TEXT NOT NULL DEFAULT '',

    -- attribution_href is the URL for the data source.
    attribution_href TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_streetlights_inserted_by ON streetlights (inserted_by);

CREATE TABLE streetlight_cells_res9 (
    id INTEGER PRIMARY KEY,
    streetlight_id TEXT NOT NULL REFERENCES streetlights(uuid) ON DELETE CASCADE,

    -- cell is the H3 cell index at resolution 9.
    cell INTEGER NOT NULL,

    UNIQUE(streetlight_id, cell)
);

CREATE INDEX idx_streetlight_cells_res9_cell ON streetlight_cells_res9 (cell);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS streetlight_cells_res9;
DROP TABLE IF EXISTS streetlights;
-- +goose StatementEnd
