<p align="center">
  <img src="frontend/src/assets/logo-bw.svg" width="128" alt="Cartomancer logo">
</p>

# Cartomancer

Your personal GPX track library.

There are a bazillion route planning and activity tracking apps, but none of them is good at managing a library of existing tracks[^1].
This one tries to be.

**⚠️ Note**: This is a work in progress and not yet ready for deployment. There are also no pre built releases, you need a (recent) Go toolchain.

## Features

- Advanced features: live weather forecasts and road closures (Switzerland only), reverse geocoding.
- Filtering, search, tagging, mark favorites.
- Map view: [SwissTopo](https://map.geo.admin.ch/).
- Except for map, does not need any live APIs. Instead, will download data in the background and query that locally (meteo and geo names).
- Easy to self host, single binary, SQLite only.

## Deployment

You may deploy the binary or the Docker image.
Configuration is possible via env vars or CLI flags, see [OPTIONS.md](OPTIONS.md).

The absolute minimum for a production deployment:

```bash
export APP_INIT_ADMIN_EMAIL=admin@example.org # Password will be printed to log once.
export APP_REGISTRATION_ENABLED=true
# Those should be persisted, otherwise sessions are lost between restarts.
export SESSION_JWT_SECRET=$(openssl rand -hex 24)
export APP_EMAIL_JWT_SECRET=$(openssl rand -hex 24)
# You may also want to configure email sending (MAIL_...).
cartomancer serve --log-pretty
```

On startup, forecast files and the geonames database will be downloaded and indexed, which hogs the database for a while.

## Development

`.envrc` contains the default dev config.
Use (direnv)[https://direnv.net/] to load it.
Otherwise, the Go toolchain and Node/npm are required.

```bash
# Starts the backend, with auto reload
direnv allow
go tool air

# In a separate shell, start the backend
cd frontend/
npm install
npm run dev
# See internal/pkg/app/config.go for login.
open http://localhost:5173
```

### Commands

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
open http://localhost:8081/jo-m.ch/go/cartomancer
```

### Email

```bash
# Run the mock email server (in a separate shell)
go tool MailHog
open http://127.0.0.1:8025/
```

### Build

The compiled frontend assets are embedded directly into the binary.
Thus, the frontend needs to be built before the backend.
`make build` will do all of that.

### Docker image

```bash
docker build -t cartomancer .
docker run -it --rm \
  -p 8080:8080 \
  --mount type=volume,src=cartomancer-data,dst=/home/nonroot/data \
  cartomancer
```

[^1]: There is https://wanderer.to/, which is quite nice. But it has many features I don't need, and is missing some I want.
