<p align="center">
  <img src="frontend/src/assets/logo-bw.svg" width="128" alt="Cartomancer logo">
</p>

# Cartomancer

The gpx track library with a touch of magic.

There are a bazillion route planning and activity tracking apps, but none of them is good at managing a library of existing tracks[^1].
This one tries to be.

**⚠️ Note**: This is a work in progress.

## Features

- Live weather forecasts and road closures (Switzerland only), reverse geocoding.
- Filtering, search, tagging, mark favorites.
- Map view: [SwissTopo](https://map.geo.admin.ch/).
- Except for map, does not need any live APIs. Instead, will download data in the background and query that locally (meteo and geo names).
- Easy to self host, single binary, SQLite only.

## Deployment

Out of pure lazyness, only Docker images are provided for [releases](https://github.com/jo-m/cartomancer/releases).
The app runs as root inside the container, it is advised to use rootless Podman/Docker.
To deploy without Docker, you need to build the binary yourself (see below).
For all configuration options, see [OPTIONS.md](OPTIONS.md).
On startup, forecast files and the geonames database will be downloaded and indexed, which will keep the CPU busy for a while.

The absolute minimum for a production deployment:

```bash
# JSONline logs, for human readable set true.
export LOG_PRETTY=false
# Create an initial admin account.
# Use setpass subcommand to set the password later.
export APP_INIT_ADMIN_EMAIL=admin@example.com
# Set to true to allow users self registration.
export APP_REGISTRATION_ENABLED=false
# Those should be persisted, otherwise sessions are lost between restarts.
export SESSION_JWT_SECRET=$(docker run ghcr.io/jo-m/cartomancer:latest genjwtsecret)
export APP_EMAIL_JWT_SECRET=$(docker run ghcr.io/jo-m/cartomancer:latest genjwtsecret)
# You may also want to configure email sending (MAIL_...).

# Run with bind mount.
mkdir -p data
docker run -it --rm                            \
  -p 8080:8080                                 \
  --mount "type=bind,src=$PWD/data,dst=/data"  \
  --env LOG_PRETTY                             \
  --env APP_INIT_ADMIN_EMAIL                   \
  --env APP_REGISTRATION_ENABLED               \
  --env SESSION_JWT_SECRET                     \
  --env APP_EMAIL_JWT_SECRET                   \
  ghcr.io/jo-m/cartomancer:latest

# To reset the admin password (adapt mount if using bind mount):
docker run -it --rm                            \
  --mount "type=bind,src=$PWD/data,dst=/data"  \
  ghcr.io/jo-m/cartomancer:latest              \
  --log-pretty                                 \
  setpass admin@example.com
```

To terminate TLS, run behind a reverse proxy.
It is a good idea to ratelimit the `/api/sessions/login` endpoint.

## Development

All build and dev tools are provided via a [Nix](https://nixos.org/) flake.
Install Nix (with flakes enabled), then either enter the devshell explicitly:

```bash
nix develop
```

or, preferred, combine with [direnv](https://direnv.net/) +
[nix-direnv](https://github.com/nix-community/nix-direnv) so entering the
repo automatically loads the shell and `.envrc` dev config:

```bash
direnv allow
```

The devshell provides Go, Node/npm, gcc, make, sqlite and goreleaser.

```bash
# Starts the backend, with auto reload
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

### Profiling (pprof)

The server includes a built-in [pprof](https://pkg.go.dev/net/http/pprof) debug server, disabled by default.
Enable it by setting a listen address:

```bash
export PPROF_ADDR=localhost:6060
```

Usage with `go tool pprof`:

```bash
# Heap profile (current allocations)
go tool pprof http://localhost:6060/debug/pprof/heap
# Heap profile (total allocations since start, useful for finding hot paths)
go tool pprof -alloc_space http://localhost:6060/debug/pprof/heap
# 30-second CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

Inside the `pprof` interactive shell, useful commands are `top`, `list <func>`, and `web` (opens a SVG graph in the browser).

### Build

The compiled frontend assets are embedded directly into the binary.
Thus, the frontend needs to be built before the backend.
`make build` will do all of that.

Reproducible builds via Nix are also provided:

```bash
# Build the binary (result/bin/cartomancer)
nix build

# Build just the frontend static assets
nix build .#frontend

# Build a minimal OCI container image as a tar.gz
nix build .#dockerImage
docker load < result
```

### Docker image

Traditional Dockerfile build:

```bash
docker build -t ghcr.io/jo-m/cartomancer:latest .
```

Or, the minimal Nix-built image (no shell, no package manager):

```bash
nix build .#dockerImage
docker load < result
```

### Creating releases

1. Go to the [GitHub Releases page](https://github.com/jo-m/cartomancer/releases) and click **Draft a new release**.
2. Create a new tag (e.g. `v1.2.0`).
3. Fill in the release title and description, then click **Publish release**.
4. The [Release workflow](.github/workflows/release.yml) will automatically build Docker images and publish a source archive to the release.

[^1]: There is https://wanderer.to/, which is quite nice. But it has many features I don't need, and is missing some I want.
