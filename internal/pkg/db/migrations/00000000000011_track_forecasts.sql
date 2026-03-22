-- +goose Up
-- +goose StatementBegin
CREATE TABLE track_forecasts (
    id INTEGER PRIMARY KEY,

    track_uuid TEXT NOT NULL REFERENCES tracks(uuid) ON DELETE CASCADE,

    created_at DATETIME NOT NULL,

    -- forecast_reference_time is the model run initialisation time used for this computation.
    forecast_reference_time DATETIME NOT NULL,

    -- start_time is the assumed journey start time.
    start_time DATETIME NOT NULL,

    -- avg_temperature_c is the average temperature along the track in degrees Celsius.
    avg_temperature_c REAL,

    -- total_precipitation_mm is the total precipitation collected over the ride duration in mm.
    total_precipitation_mm REAL,

    -- Wind rose values: average wind speed (m/s) in each of the 4 relative-to-track sectors.
    -- Head means the wind is blowing against the direction of travel.
    wind_head_ms REAL,
    wind_right_ms REAL,
    wind_tail_ms REAL,
    wind_left_ms REAL,

    UNIQUE(track_uuid)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE track_forecasts;
-- +goose StatementEnd
