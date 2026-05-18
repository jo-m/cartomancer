-- +goose Up
-- +goose StatementBegin
CREATE TABLE forecasts (
    id INTEGER PRIMARY KEY,

    created_at DATETIME NOT NULL,

    -- reference_time is the model run initialisation time.
    reference_time DATETIME NOT NULL UNIQUE,

    -- Bounding box of the spatial domain covered by this run (WGS84 degrees).
    bounds_min_lat REAL,
    bounds_min_lon REAL,
    bounds_max_lat REAL,
    bounds_max_lon REAL,

    -- horizontal_grid_file holds the raw GRIB2 horizontal grid constants file content.
    horizontal_grid_file BLOB NOT NULL,

    -- vertical_grid_file holds the raw GRIB2 vertical grid constants file content.
    vertical_grid_file BLOB NOT NULL,

    -- attribution is the human-readable data source credit.
    attribution TEXT NOT NULL DEFAULT '',

    -- attribution_href is the URL for the data source.
    attribution_href TEXT NOT NULL DEFAULT ''
);

-- forecast_files holds per-(forecast, variable, valid_time) metadata only.
-- The associated GRIB2 payload lives in forecast_file_blobs, keyed by id, so
-- that scans over metadata (admin listings, window queries) do not have to
-- touch the leaf pages that hold multi-MB blob records.
CREATE TABLE forecast_files (
    id INTEGER PRIMARY KEY,

    -- valid_time is the time at which the forecast values in this file apply
    -- (horizon is the duration from reference_time to valid_time).
    valid_time DATETIME NOT NULL,

    -- valid_until_time is the exclusive upper bound of the validity interval
    -- [valid_time, valid_until_time).
    valid_until_time DATETIME NOT NULL,

    -- variable is the forecast variable name, e.g. 'U_10M', 'V_10M', 'TOT_PREC'.
    variable TEXT NOT NULL,

    -- file_size is the length in bytes of the associated blob in
    -- forecast_file_blobs.file. Stored here so listings can read sizes
    -- without touching the blob table.
    file_size INTEGER NOT NULL,

    forecast_id INTEGER NOT NULL REFERENCES forecasts(id) ON DELETE CASCADE,

    UNIQUE(forecast_id, variable, valid_time)
);

-- forecast_file_blobs stores the raw GRIB2 payload separately so that the
-- metadata b-tree above stays small and dense. The PK is the same id as the
-- parent metadata row; ON DELETE CASCADE keeps the two in sync automatically.
CREATE TABLE forecast_file_blobs (
    forecast_file_id INTEGER PRIMARY KEY
        REFERENCES forecast_files(id) ON DELETE CASCADE,

    -- file holds the raw GRIB2 file content.
    file BLOB NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS forecast_file_blobs;
DROP TABLE IF EXISTS forecast_files;
DROP TABLE IF EXISTS forecasts;
-- +goose StatementEnd
