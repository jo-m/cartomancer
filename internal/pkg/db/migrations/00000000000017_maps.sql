-- +goose Up
CREATE TABLE map_builds (
    uuid TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,

    -- Fields from the protomaps builds.json metadata.
    key TEXT NOT NULL,
    size INTEGER NOT NULL,
    md5sum TEXT NOT NULL,
    uploaded DATETIME NOT NULL,
    version TEXT NOT NULL,

    -- Extraction parameters used to produce the local file.
    -- All four bbox columns are NULL (entire world) or all four are set.
    maxzoom INTEGER NOT NULL,
    bbox_min_lon REAL,
    bbox_min_lat REAL,
    bbox_max_lon REAL,
    bbox_max_lat REAL,

    -- Filename in the maps directory is <uuid>.pmtiles.
    -- This column records whether the extraction completed successfully.
    ready INTEGER NOT NULL DEFAULT 0,

    CHECK (
        (bbox_min_lon IS NULL AND bbox_min_lat IS NULL AND bbox_max_lon IS NULL AND bbox_max_lat IS NULL)
        OR
        (bbox_min_lon IS NOT NULL AND bbox_min_lat IS NOT NULL AND bbox_max_lon IS NOT NULL AND bbox_max_lat IS NOT NULL)
    )
);

-- +goose Down
DROP TABLE IF EXISTS map_builds;
