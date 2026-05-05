# Backend

## Package structure

```
internal/pkg/...
- api            REST API endpoints and handlers (Chi)
- app            App-level config
- attribute      Standard attrib struct for licensed data sources
- blob           Blob storage for GPX/FIT files
- db             Main SQLite database: connection, migrations, sqlc-generated queries
   - forecastdb  Separate SQLite database for forecast data, regenerable
   - geonamesdb  Separate SQLite database for geonames data, regenerable
- forecast       Loads weather forecast data, point sampling by variable/time/location
- geoadmin       Client for data.geo.admin.ch STAC API
- geonames       Reverse geocoding
- grib2          Minimal parser for GRIB2 meteorological data format
- jobs           Persistent async job queue
- load           GPX/FIT file parsing → TrackSource
- logg           Structured logging (slog) and helpers
- mail           Email job handler
- maps           PMTiles map tiles, protomaps API client
- meteo          Downloads ICON-CH1-EPS weather forecast data
- password       Password gen/hashing
- roadclosures   Fetches road closures and detours
- segment        Extracts shared road segments from tracks using H3 cell clustering
- session        JWT+cookie session management, middleware
- track          Track types, enums, metadata calculations
- trackgroup     Groups similar tracks by comparing H3 cell paths
- users          OTP (TOTP/HOTP) support
- utl            General utilities
```

## REST API

Handlers are mounted in internal/pkg/api/api.go.
Group handler funcs into files, ca. 1 per resource, in internal/pkg/api/*.go.
They all must be methods on the Server struct.
internal/pkg/api/openapi.yaml MUST be updated when ever endpoints change.
Follow RESTful API design guidelines, and use appropriate HTTP methods and status codes.
Use camelCase for any JSON fields (e.g. SessionID -> sessionId).
Use the helpers in internal/pkg/api/error.go.
In most cases where an error is returned from a handler, the details should be logged.
Caching: Endpoints which seldomly change and are expensive to compute must include etag handling and cache headers. Example: handleDownloadTrackSVG().

## Job queue

Async jobs are persisted in SQLite and executed by a worker pool (`internal/pkg/jobs/`).

To add a new job type:
1. Define an args struct implementing `Kind() string`
2. Implement a `jobs.Job[MyArgs]` handler
3. Register with `jobs.MustRegisterJob(workers, &MyHandler{})`
4. Submit with `jobs.Submit(ctx, submitter, MyArgs{...}, jobs.Params{})`

At-least-once semantics; configure retries via `jobs.Params{MaxRetries: N}`.

## Linting and code quality

After every change, `make check` MUST run successfully.
This already includes `go build ./...`.

ALWAYS go through the `Makefile` for formatting/linting/testing.
NEVER invoke any tool directly.

- make format: gofmt + go mod tidy
- make lint: full lint suite (mod tidy, gofmt, vet, staticcheck, revive, govulncheck, gosec)
- make test: go test ./... with non-failure output filtered
- make check: gen + lint + go build ./... + go test ./...
- make gen: regenerate sqlc code (required after query changes)

NEVER use `go build`, use `go run` directly to run binaries from this repo.

## Testing

Use db.GetTestDB(t), geonamesdb.GetTestDB(t), or forecastdb.GetTestDB(t) to get a temp SQLite DB with all migrations applied.
Use github.com/stretchr/testify/require for assertions.
Use github.com/franiglesias/golden for snapshot tests. Approval mode: golden.Verify(t, output, golden.WaitApproval()).
Set custom extension for snapshot files: golden.Verify(t, output, golden.Extension(".json")).

## Conventions

- All files with ending `.gen.go` are generated and MUST NOT EVER be edited manually. You should also not read them manually, instead use grep or LSP plugin.
- The logger instance is usually passed around in ctx.Context.
- Log messages are generally lower case and without punctuation.
- Avoid TOCTOU race conditions by using txs correctly. Be careful to hold txs only for a short time.
- All public fns must have docstrings.
- All code comments must be grammatical complete sentences and end with punctuation (interjections are grammatically also complete sentences).
- MUST write tests for all new code

## Config structs

Make modules/packages have their own config structs if applicable, compatible with github.com/alexflint/go-arg.
Existing ones can be found by grepping for `Config struct {`.

- Example: `internal/pkg/app/config.go`
- Must have a consistent prefix for args and env vars
- Mention github.com/alexflint/go-arg in the docstring, See AppConfig
- Must have a Validate() fn, errors must mention the arg and env var name
