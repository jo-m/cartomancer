<p align="center">
  <img src="frontend/src/assets/logo-bw.svg" width="128" alt="Cartomancer logo">
</p>

# Cartomancer

The track library with a touch of magic.

There are a bazillion route planning and activity tracking apps, but none of them is good at managing a library of existing tracks[^1].
This one tries to be.

**⚠️ Note**: This is a work in progress.

## Features

- Advanced features: live weather forecasts and road closures (Switzerland only), reverse geocoding.
- Filtering, search, tagging, mark favorites.
- Map view: [SwissTopo](https://map.geo.admin.ch/).
- Except for map, does not need any live APIs. Instead, will download data in the background and query that locally (meteo and geo names).
- Easy to self host, single binary, SQLite only.

## Deployment

Out of pure lazyness, only Docker images are provided for [releases](https://github.com/jo-m/cartomancer/releases).
To deploy without Docker, you need to build the binary yourself (see below).
For all configuration options, see [OPTIONS.md](OPTIONS.md).
On startup, forecast files and the geonames database will be downloaded and indexed, which will keep the CPU busy for a while.

The absolute minimum for a production deployment:

```bash
export LOG_PRETTY=false # JSONline logs, for human readable set true.
export APP_INIT_ADMIN_EMAIL=admin@example.org # Password will be printed to log once.
export APP_REGISTRATION_ENABLED=true
# Those should be persisted, otherwise sessions are lost between restarts.
export SESSION_JWT_SECRET=$(docker run ghcr.io/jo-m/cartomancer:latest genjwtsecret)
export APP_EMAIL_JWT_SECRET=$(docker run ghcr.io/jo-m/cartomancer:latest genjwtsecret)
# You may also want to configure email sending (MAIL_...).

docker run -it --rm                                               \
  -p 8080:8080                                                    \
  --mount type=volume,src=cartomancer-data,dst=/home/nonroot/data \
  --env LOG_PRETTY                                                \
  --env APP_INIT_ADMIN_EMAIL                                      \
  --env APP_REGISTRATION_ENABLED                                  \
  --env SESSION_JWT_SECRET                                        \
  --env APP_EMAIL_JWT_SECRET                                      \
  ghcr.io/jo-m/cartomancer:latest                                 \
  serve
```

## Development

`.envrc` contains the default dev config.
Use (direnv)[https://direnv.net/] to load it.
Otherwise, the Go toolchain and Node/npm are required.

```bash
# Starts the backend, with auto reload
direnv allow
go tool air

# In a separate shell, start the frontend
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
```

### Creating releases

1. Go to the [GitHub Releases page](https://github.com/jo-m/cartomancer/releases) and click **Draft a new release**.
2. Create a new tag (e.g. `v1.2.0`).
3. Fill in the release title and description, then click **Publish release**.
4. The [Release workflow](.github/workflows/release.yml) will automatically build Docker images and publish a source archive to the release.

[^1]: There is https://wanderer.to/, which is quite nice. But it has many features I don't need, and is missing some I want.
