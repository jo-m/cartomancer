-- +goose Up
-- +goose StatementBegin
CREATE TABLE geonames (
    geonameid INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    asciiname TEXT NOT NULL DEFAULT '',
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
    population INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_geonames_reverse
    ON geonames (feature_class, latitude, longitude);

CREATE INDEX idx_geonames_country
    ON geonames (country_code, feature_class);

-- FTS5 index for prefix search on place names.
-- External-content table: data lives in geonames, FTS holds only the index.
-- unicode61 tokenizer handles case folding and diacritic removal.
CREATE VIRTUAL TABLE geonames_fts USING fts5(
    name, asciiname,
    content=geonames, content_rowid=geonameid,
    tokenize='unicode61 remove_diacritics 2'
);

-- Populate the FTS index from any existing geonames data.
INSERT INTO geonames_fts(geonames_fts) VALUES('rebuild');

-- First-level administrative divisions (states, provinces, etc.).
CREATE TABLE geoname_admin1 (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    geonameid INTEGER NOT NULL
) WITHOUT ROWID;

-- Second-level administrative divisions (counties, districts, etc.).
CREATE TABLE geoname_admin2 (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    geonameid INTEGER NOT NULL
) WITHOUT ROWID;

-- Generated geoname labels for tracks.
CREATE TABLE track_geonames (
    track_id TEXT PRIMARY KEY REFERENCES tracks(uuid) ON DELETE CASCADE,
    label TEXT NOT NULL,
    created_at DATETIME NOT NULL
) WITHOUT ROWID;

-- Tracks when geonames data was last imported.
CREATE TABLE geoname_imports (
    id INTEGER PRIMARY KEY,
    created_at DATETIME NOT NULL,
    row_count INTEGER NOT NULL DEFAULT 0
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS track_geonames;
DROP TABLE IF EXISTS geoname_imports;
DROP TABLE IF EXISTS geoname_admin2;
DROP TABLE IF EXISTS geoname_admin1;
DROP TABLE IF EXISTS geonames_fts;
DROP TABLE IF EXISTS geonames;
-- +goose StatementEnd
