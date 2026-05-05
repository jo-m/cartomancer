## Database

### Split database architecture

There are three separate SQLite databases, all stored in the data directory (`--data-dir`):
- db.sqlite: main database (users, tracks, sessions, jobs).
- geonamesdb.sqlite: geonames geographical data. Regenerable, do not back up.
- forecast.sqlite: weather forecast data. Regenerable, do not back up.

Each db has its own pkg under internal/pkg/db/ with independent migrations and sqlc config.
This gives independent write locks so bulk-imports on geonames or forecasts do not block the main db.

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
- rw: read/write, max 1 conn
- ro: read-only, connection pool

WAL mode, foreign keys enabled, busy timeout 5s. Because each database has its own connection pool, write locks are independent.

### Transactions

```
// auto commit/rollback
d.WithTx(ctx, func(tx *db.Queries) error { ... })  
// manual
tx, err := d.BeginTX(ctx)                          
```

### Transactions in REST handlers

Use db.WithTx() whenever a handler performs a guard check followed by a mutation, to prevent TOCTOU races.
Example: fetching a row to check permissions and then deleting/updating it must be a single tx.

Use sentinel errors to distinguish business-logic failures (which map to 4xx) from unexpected errors inside `WithTx`:

```
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

Note: session.Create and session.Delete open their own WithTx internally and cannot be nested inside another WithTx.
Perform those calls before or after the handler's own transaction.

### Queries

No ORM.
Written in SQL in `internal/pkg/db/queries/*.sql` (main DB), `internal/pkg/db/geonamesdb/queries/*.sql`, and `internal/pkg/db/forecastdb/queries/*.sql`.
Compiled to Go methods via sqlc.
Run `make gen` after editing queries.
When querying datetimes, you MUST use the sqlite `datetime()` function on both sides to account for timezones etc.
We almost NEVER want to join data in Go code - instead use JOIN statements to let SQLite do the work.
