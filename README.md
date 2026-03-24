A personal GPX tracks library.
There are a bazillion route planning and activity tracking apps, but none of them is good at managing a library of existing tracks[^1].
This one tries to be.

# Features

- Advanced features: live weather forecasts and road closures (Switzerland only), reverse geocoding.
- Filtering, search, tagging, mark favorites.
- Map view: [SwissTopo](https://map.geo.admin.ch/).
- Except for map, does not need any live APIs. Instead, will download data in the background and query that locally (meteo and geo names).
- Easy to self host, single binary, SQLite only.

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

# View Go docs
go run golang.org/x/pkgsite/cmd/pkgsite@latest -open -http localhost:8081
open http://localhost:8081/jo-m.ch/go/detour
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

[^1]: There is https://wanderer.to/, which is quite nice. But it has many features I don't need, and is missing some I want.
