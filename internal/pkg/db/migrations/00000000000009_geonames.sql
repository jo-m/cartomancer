-- +goose Up
-- +goose StatementBegin
CREATE TABLE geonames (
    geonameid INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    asciiname TEXT NOT NULL DEFAULT '',
    alternatenames TEXT NOT NULL DEFAULT '',
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    feature_class TEXT NOT NULL DEFAULT '',
    feature_code TEXT NOT NULL DEFAULT '',
    country_code TEXT NOT NULL DEFAULT '',
    cc2 TEXT NOT NULL DEFAULT '',
    admin1_code TEXT NOT NULL DEFAULT '',
    admin2_code TEXT NOT NULL DEFAULT '',
    admin3_code TEXT NOT NULL DEFAULT '',
    admin4_code TEXT NOT NULL DEFAULT '',
    population INTEGER NOT NULL DEFAULT 0,
    elevation INTEGER,
    dem INTEGER NOT NULL DEFAULT 0,
    timezone TEXT NOT NULL DEFAULT '',
    modification_date TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_geonames_reverse
    ON geonames (feature_class, latitude, longitude);

CREATE INDEX idx_geonames_country
    ON geonames (country_code, feature_class);

-- Tracks when geonames data was last imported.
CREATE TABLE geoname_imports (
    id INTEGER PRIMARY KEY,
    created_at DATETIME NOT NULL,
    row_count INTEGER NOT NULL DEFAULT 0
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS geoname_imports;
DROP TABLE IF EXISTS geonames;
-- +goose StatementEnd
