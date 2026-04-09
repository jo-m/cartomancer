-- +goose Up
-- Move geonames and forecast tables to separate database files.
-- Data in these tables is regenerated from external sources and does not need
-- to be preserved; the separate databases will recreate the tables on startup.
-- +goose StatementBegin

-- Geonames tables (now in geonamesdb).
DROP TABLE IF EXISTS geonames_fts;
DROP TABLE IF EXISTS geonames;
DROP TABLE IF EXISTS geoname_admin1;
DROP TABLE IF EXISTS geoname_admin2;
DROP TABLE IF EXISTS geoname_imports;

-- Forecast tables (now in forecastdb).
DROP TABLE IF EXISTS forecast_files;
DROP TABLE IF EXISTS forecasts;
-- +goose StatementEnd

-- +goose Down
-- Re-creating original tables is not supported; the separate databases
-- hold this data now. A downgrade requires restoring from backup.
