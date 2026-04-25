# Backend

## Package structure

```
internal/pkg/
├── api/            # HTTP REST API endpoints and handlers (Chi router)
├── app/            # Application-level config
├── attribute/      # Standard TASL attribution struct for CC-licensed data sources
├── blob/           # Blob storage for GPX/FIT files (zstd-compressed, SQLite)
├── db/             # Main SQLite database: connection, migrations, sqlc-generated queries
│   ├── forecastdb/ # Separate SQLite database for forecast data (regenerable, excludable from backups)
│   └── geonamesdb/ # Separate SQLite database for geonames data (regenerable, excludable from backups)
├── forecast/       # Loads GRIB2 weather forecast data, point sampling by variable/time/location
├── geoadmin/       # Client for Swiss government STAC API (data.geo.admin.ch)
├── geonames/       # Reverse geocoding via GeoNames geographical database
├── grib2/          # Minimal parser for GRIB2 binary meteorological data format
├── jobs/           # Persistent async job queue
├── load/           # GPX/FIT file parsing → TrackSource
├── logg/           # Structured logging (slog), middleware, context helpers
├── mail/           # Email job handler (SMTP via go-mail)
├── maps/           # PMTiles map tile extraction: config, downloader job, cleaner job, protomaps API client
├── meteo/          # Downloads ICON-CH1-EPS weather forecast data from Swiss STAC API
├── password/       # Argon2id hashing
├── roadclosures/   # Fetches bike road closures and detours from geo.admin.ch
├── segment/        # Extracts shared road segments from tracks using H3 cell clustering
├── session/        # JWT+cookie session management, middleware
├── track/          # Track types, enums, metadata calculations
├── trackgroup/     # Groups similar tracks by comparing H3 cell paths
├── users/          # OTP (TOTP/HOTP) support
└── utl/            # General utilities
```

### REST API

Handlers are mounted in `internal/pkg/api/api.go`.
Group handler fns into files, approx. 1 per resource, in `internal/pkg/api/*.go`.
They all must be methods on the `Server` struct.
`internal/pkg/api/openapi.yaml` MUST be updated when ever endpoints change.
Follow RESTful API design guidelines, and use appropriate HTTP methods and status codes.
Use camelCase for any JSON fields (e.g. "SessionID string `json:"sessionId"`").
Use the helpers in `internal/pkg/api/error.go`.
In most cases where an error is returned from a handler, the details should be logged.
Caching: Endpoints which seldomly change and are expensive to compute should include aggressive caching/etag headers. Example: `handleDownloadTrackSVG()`.

### Middleware stack (applied in order in main.go)

1. `chi.middleware.RequestID`
2. `chi.middleware.ThrottleBacklog` — limits concurrent requests (if configured)
3. `logg.AttachLogger` — attaches logger with request ID to context
4. `logg.RequestLogger` — logs each request with duration/status
5. `chi.middleware.RequestSize(5MB)`
6. `chi.middleware.Compress(5)`
7. `sess.Middleware` — auto-creates/loads session for every request
8. `chi.middleware.Recoverer`

### Context access in handlers

```go
logg.Debug(ctx, "msg", "key", val)   // logger from context
logg.Error(ctx, "msg", "err", err)

session.MustGet(ctx)                 // current session (always set)
session.GetUser(ctx)                 // *db.User, nil if anonymous
```

## Database

### Split database architecture

There are three separate SQLite databases, all stored in the data directory (`--data-dir`):
- `db.sqlite` — main database (users, tracks, sessions, jobs). Back this up.
- `geonamesdb.sqlite` — geonames geographical data. Regenerable, excludable from backups.
- `forecast.sqlite` — weather forecast data. Regenerable, excludable from backups.

Each database has its own package under `internal/pkg/db/` with independent migrations, sqlc config, and connection pools. This gives independent write locks so bulk-importing geonames or downloading forecasts does not block the main database.

Cross-database references (`track_geonames`, `track_forecasts`) remain in the main database with FK to tracks; they store only the association, not the heavy data.

### Migrations

github.com/pressly/goose/v3. Main DB migrations are in `internal/pkg/db/migrations/*.sql`, sub-databases in `internal/pkg/db/{geonamesdb,forecastdb}/migrations/*.sql`.
To create a new one (main DB):
```bash
go tool goose status
go tool goose create REPLACEME sql
go tool goose up
go tool goose validate
```
Most tables must use `uuid TEXT PRIMARY KEY`, only internal ones can use `id INTEGER PRIMARY KEY`. Populated with uuid.NewV7().
UUID v7 values are monotonically increasing (time-ordered), so they can be used as cursors for keyset/cursor-based pagination (e.g. `WHERE uuid > ? ORDER BY uuid ASC LIMIT ?`).
Track points (lat/lon/elevation stored in blobs) are immutable once inserted.
DO NOT explicitly annotate cols with NULL if they are nullable, otherwise sqlc will emit interface{} (NULL is implicit anyways if left out).
Always store timestamps with timezone.
Simple internal enums, where the logic/state is managed from within SQL queries only, are strings with CHECK, example: `status TEXT CHECK(status IN ('C', 'R', 'A', 'E', 'S') ) NOT NULL DEFAULT 'C'`.
"App" enums which are managed from Go code are integers, declared with iota.
Most tables have created_at, updated_at, created_by.
IMPORTANT: When adding or changing columns in existing tables, you MUST ask the operator first if an existing migration should be edited, or a new one created.
IMPORTANT: When adding new columns to the `users` or `email_verifications` tables, check whether those columns need to be included in the demo mode trigger lockdown in `internal/pkg/app/demo.go` (`demo_users_no_update` trigger's WHEN clause).

### Connection model

Each database (main, geonamesdb, forecastdb) opens two connections (`internal/pkg/db/open.go`):
- `rw` — read/write, max 1 connection (SQLite requirement)
- `ro` — read-only, connection pool

WAL mode, foreign keys enabled, busy timeout 5s. Because each database has its own connection pool, write locks are independent.

### Transactions

```go
d.WithTx(ctx, func(tx *db.Queries) error { ... })  // auto commit/rollback
tx, err := d.BeginTX(ctx)                          // manual
```

### Transactions in REST handlers

Use `WithTx` whenever a handler performs a guard check followed by a mutation, to prevent TOCTOU races.
Example: fetching a row to check permissions and then deleting/updating it must be a single tx.

Use sentinel errors to distinguish business-logic failures (which map to 4xx) from unexpected errors inside `WithTx`:

```go
var errSomething = errors.New("...")

err := sv.d.WithTx(ctx, func(q *db.Queries) error {
    row, txErr := q.GetFoo(ctx, id)
    if txErr != nil { return txErr }
    if row.SomeCondition { return errSomething }
    _, txErr = q.DeleteFoo(ctx, id)
    return txErr
})
if errors.Is(err, sql.ErrNoRows) { writeError(w, 404, "not found"); return }
if errors.Is(err, errSomething) { writeError(w, 409, "..."); return }
if err != nil { logg.Error(...); writeError(w, 500, "..."); return }
```

Note: `session.Create` and `session.Delete` open their own `WithTx` internally and cannot be nested inside another `WithTx`. Perform those calls before or after the handler's own transaction.

### Queries

No ORM.
Written in SQL in `internal/pkg/db/queries/*.sql` (main DB), `internal/pkg/db/geonamesdb/queries/*.sql`, and `internal/pkg/db/forecastdb/queries/*.sql`.
Compiled to Go methods via sqlc.
Run `make gen` after editing queries.
When querying datetimes, you MUST use the sqlite `datetime()` function on both sides to account for timezones etc.

## Job queue

Async jobs are persisted in SQLite and executed by a worker pool (`internal/pkg/jobs/`).

To add a new job type:
1. Define an args struct implementing `Kind() string`
2. Implement a `jobs.Job[MyArgs]` handler
3. Register with `jobs.MustRegisterJob(workers, &MyHandler{})`
4. Submit with `jobs.Submit(ctx, submitter, MyArgs{...}, jobs.Params{})`

At-least-once semantics; configure retries via `jobs.Params{MaxRetries: N}`.

## Linting and code quality

After every change, `make check` MUST run successfully. This already includes `go build ./...`.

## Looking up Go APIs

When exploring unknown APIs, or even looking up APIs internal to the project, use `go doc -short <pkg>` (overview) and `go doc -all <pkg>` (detailed).
Only read the full source code if you need detailed understanding of the implementation.

## Development

```bash
air        # Hot-reload (pre-build runs make gen, watches .go/.sql)
make gen   # Regenerate sqlc code (required after query changes)
make check # Full quality gate
```

NEVER use `go build`, use `go run` directly to run binaries from this repo.

For email, run the bundled MailHog: `go tool MailHog` (UI at http://127.0.0.1:8025).

## Testing

Use `db.GetTestDB(t)`, `geonamesdb.GetTestDB(t)`, or `forecastdb.GetTestDB(t)` to get a temp SQLite DB with all migrations applied.
Use `github.com/stretchr/testify/require` for assertions.
Use `github.com/franiglesias/golden` for snapshot tests. Approval mode: `golden.Verify(t, output, golden.WaitApproval())`. Set custom extension for snapshot files: golden.Verify(t, output, golden.Extension(".json")).

## Conventions

- All files with ending `.gen.go` are generated and MUST NOT EVER be edited manually. You should also not read them manually, instead use grep or LSP plugin.
- The logger instance is mostly passed around in ctx.Context.
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
