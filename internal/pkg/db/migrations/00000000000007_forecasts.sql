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

    -- grid_file holds the raw GRIB2 horizontal grid constants file content.
    grid_file BLOB NOT NULL
);

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

    -- file holds the raw GRIB2 file content.
    file BLOB NOT NULL,

    forecast_id INTEGER NOT NULL REFERENCES forecasts(id),

    UNIQUE(forecast_id, variable, valid_time)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE forecast_files;
DROP TABLE IF EXISTS forecasts;
-- +goose StatementEnd
