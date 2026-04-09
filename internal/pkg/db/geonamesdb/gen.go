// Package geonamesdb contains migrations, models, queries and DB connections
// for the GeoNames geographical database. This data lives in a separate SQLite
// file so it can be excluded from backups and does not block writes on the main
// database during bulk imports.
package geonamesdb

//go:generate go tool sqlc generate
//go:generate go tool sqlc vet
