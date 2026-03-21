A personal GPX tracks library.

# Features

- Easy to self host, single binary, SQLite only
- Weather forecasts for tracks (currently only [MeteoSwiss](https://opendatadocs.meteoswiss.ch/de/))
- Map view (currently only [SwissTopo](https://map.geo.admin.ch/))
- Reverse geocoding
- Except for map, does not need any live APIs. Instead, will download data and query that locally (meteo and geo names).

# Development

`.envrc` contains the default dev config.
Use (direnv)[https://direnv.net/] to load it.

```bash
# Starts the backend, with auto reload
direnv allow
go tool air

# In a separate shell, start the backend
cd frontend/
npm run dev
# See internal/pkg/app/config.go for login.
open http://localhost:5173
```

## Commands

```bash
# See Makefile for more.
make gen
make test
make test_online
make check

# Migrations
go tool goose status
go tool goose create REPLACEME sql
go tool goose up
go tool goose validate

# SQL queries
go tool sqlc generate
go tool sqlc vet --file internal/pkg/db/sqlc.yaml
```

## Email

```bash
# Run the mock emails server (in a separate shell)
go tool MailHog
open http://127.0.0.1:8025/
```

## Docker

```bash
docker build -t detour .
docker run -it --rm -p 8080:8080 --mount type=volume,src=detour-data,dst=/home/nonroot/data detour
```
