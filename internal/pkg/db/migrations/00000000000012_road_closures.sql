-- +goose Up
-- +goose StatementBegin
CREATE TABLE road_closures (
    uuid TEXT PRIMARY KEY,

    -- source_id is the identifier from the upstream data source (e.g. featureId).
    -- Not necessarily numeric; format depends on the source.
    source_id TEXT NOT NULL,

    -- inserted_by identifies which job inserted this row, so that each job can
    -- delete only its own rows during a refresh cycle.
    inserted_by TEXT NOT NULL,

    created_at DATETIME NOT NULL,

    -- type is the closure kind from the source (e.g. "detour", "closed_way").
    type TEXT NOT NULL,

    starts_at DATETIME,
    ends_at DATETIME,
    title TEXT NOT NULL,
    reason TEXT,
    description TEXT,
    content_provider TEXT,

    -- geometry is the GeoJSON geometry object stored as a JSON text string.
    geometry TEXT NOT NULL,

    -- attribution is the human-readable data source credit.
    attribution TEXT NOT NULL DEFAULT '',

    -- attribution_href is the URL for the data source.
    attribution_href TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_road_closures_inserted_by ON road_closures (inserted_by);

CREATE TABLE road_closure_cells_res7 (
    id INTEGER PRIMARY KEY,
    road_closure_id TEXT NOT NULL REFERENCES road_closures(uuid) ON DELETE CASCADE,

    -- cell is the H3 cell index at resolution 7.
    cell INTEGER NOT NULL,

    UNIQUE(road_closure_id, cell)
);

CREATE INDEX idx_road_closure_cells_res7_cell ON road_closure_cells_res7 (cell);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS road_closure_cells_res7;
DROP TABLE IF EXISTS road_closures;
-- +goose StatementEnd
