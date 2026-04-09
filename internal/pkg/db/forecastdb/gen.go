// Package forecastdb contains migrations, models, queries and DB connections
// for weather forecast data (GRIB2 files). This data lives in a separate SQLite
// file so it can be excluded from backups and does not block writes on the main
// database during forecast downloads.
package forecastdb

//go:generate go tool sqlc generate
//go:generate go tool sqlc vet
