-- +goose Up
-- +goose StatementBegin
CREATE TABLE forecast_files (
    id INTEGER PRIMARY KEY,

    created_at DATETIME NOT NULL,

    -- reference_time is the model run initialisation time.
    reference_time DATETIME NOT NULL,
    -- valid_time is the time at which the forecast values apply.
    valid_time DATETIME NOT NULL,

    -- variable is the forecast variable name, e.g. 'U_10M', 'V_10M', 'TOT_PREC'.
    variable TEXT NOT NULL,

    -- Bounding box of the spatial domain covered by this file (WGS84 degrees).
    bounds_min_lat REAL,
    bounds_min_lon REAL,
    bounds_max_lat REAL,
    bounds_max_lon REAL,

    blob_id INTEGER NOT NULL,

    FOREIGN KEY(blob_id) REFERENCES blobs(id) ON DELETE RESTRICT,
    UNIQUE(reference_time, variable, valid_time)
);
-- When a forecast_file is deleted, delete its blob (unless another forecast_file still references it).
CREATE TRIGGER forecast_files_delete_blob AFTER DELETE ON forecast_files
BEGIN
    DELETE FROM blobs WHERE id = OLD.blob_id
    AND NOT EXISTS (SELECT 1 FROM forecast_files WHERE blob_id = OLD.blob_id);
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER forecast_files_delete_blob;
DROP TABLE forecast_files;
-- +goose StatementEnd
