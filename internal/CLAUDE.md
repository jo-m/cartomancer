# Backend

## Package structure

```
internal/pkg/...
- api            REST endpoints/handlers (Chi)
- app            App-level config
- attribute      Standard attrib struct for licensed data sources
- blob           Blob storage for GPX/FIT files
- db             Main SQLite DB: connection, migrations, sqlc queries
   - forecastdb  Separate SQLite DB for forecast data, regenerable
   - geonamesdb  Separate SQLite DB for geonames data, regenerable
- forecast       Loads weather forecast, point sampling by var/time/loc
- geoadmin       Client for data.geo.admin.ch STAC API
- geonames       Reverse geocoding
- grib2          Minimal parser for GRIB2 meteorological format
- jobs           Persistent async job queue
- load           GPX/FIT file parsing -> TrackSource
- logg           Structured logging (slog) and helpers
- mail           Email job handler
- maps           PMTiles map tiles, protomaps API client
- meteo          Downloads ICON-CH1-EPS weather forecast data
- password       Password gen/hashing
- roadclosures   Fetches road closures and detours
- segment        Extracts shared road segments via H3 cell clustering
- session        JWT+cookie session management, middleware
- track          Track types, enums, metadata calculations
- trackgroup     Groups similar tracks by comparing H3 cell paths
- users          OTP (TOTP/HOTP) support
- utl            General utilities
```

## REST API

- Handlers mounted in `internal/pkg/api/api.go`; group by resource (~1 file each) under `internal/pkg/api/*.go`. All must be methods on `Server`.
- `internal/pkg/api/openapi.yaml` MUST be updated when endpoints change.
- Follow REST conventions; use appropriate HTTP methods/status codes.
- JSON fields are camelCase (e.g. `SessionID` -> `sessionId`).
- Use helpers in `internal/pkg/api/error.go`. Log error details when returning errors from handlers.
- Caching: endpoints that seldom change and are expensive must use etag + cache headers (see `handleDownloadTrackSVG()`).

## Job queue

Async jobs persisted in SQLite, executed by a worker pool (`internal/pkg/jobs/`). To add a job type:
1. Define an args struct implementing `Kind() string`.
2. Implement a `jobs.Job[MyArgs]` handler.
3. `jobs.MustRegisterJob(workers, &MyHandler{})`.
4. `jobs.Submit(ctx, submitter, MyArgs{...}, jobs.Params{})`.

At-least-once semantics; configure retries via `jobs.Params{MaxRetries: N}`.

## Linting and code quality

After every change, `make check` MUST succeed (it includes `go build ./...`).
ALWAYS go through the `Makefile`; NEVER invoke linters/formatters directly.

- `make format` - gofmt + go mod tidy
- `make lint`   - mod tidy, gofmt, vet, staticcheck, revive, govulncheck, gosec
- `make test`   - `go test ./...` with non-failure output filtered
- `make check`  - gen + lint + build + test
- `make gen`    - regenerate sqlc code (required after query changes)

NEVER use `go build`; use `go run` to run binaries from this repo.

## Testing

- `db.GetTestDB(t)` / `geonamesdb.GetTestDB(t)` / `forecastdb.GetTestDB(t)` for a temp SQLite with all migrations applied.
- Assertions: `github.com/stretchr/testify/require`.
- Snapshot tests: `github.com/franiglesias/golden`. Approval mode: `golden.Verify(t, output, golden.WaitApproval())`. Custom extension: `golden.Verify(t, output, golden.Extension(".json"))`.

## Conventions

- `*.gen.go` is generated: NEVER edit or read manually (use grep/LSP).
- Logger is passed via `ctx.Context`. Log messages: lowercase, no punctuation.
- Avoid TOCTOU: use txs correctly and hold them briefly.
- All public fns need docstrings.
- Code comments: grammatical complete sentences ending in punctuation.
- MUST write tests for all new code.

## Config structs

Per-module config structs compatible with `github.com/alexflint/go-arg`. Find existing ones via `Config struct {`.

- Example: `internal/pkg/app/config.go`.
- Consistent prefix for args and env vars.
- Reference `github.com/alexflint/go-arg` and `See AppConfig` in the docstring.
- Must have `Validate()`; errors must mention the arg and env var name.
