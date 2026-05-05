## Database

### Split database architecture

Three separate SQLite DBs in `--data-dir`:
- `db.sqlite`         - main (users, tracks, sessions, jobs).
- `geonamesdb.sqlite` - geonames data. Regenerable, do not back up.
- `forecast.sqlite`   - weather forecast data. Regenerable, do not back up.

Each has its own pkg under `internal/pkg/db/` with independent migrations and sqlc config. Independent write locks mean bulk imports on geonames/forecasts do not block the main DB.

Cross-DB references (`track_geonames`, `track_forecasts`) live in the main DB with FK to tracks; they store only the association, not the heavy data.

### Migrations

`github.com/pressly/goose/v3`. Main DB: `internal/pkg/db/migrations/*.sql`; sub-DBs: `internal/pkg/db/{geonamesdb,forecastdb}/migrations/*.sql`.

Create a main-DB migration:
```bash
go tool goose status
go tool goose create REPLACEME sql
go tool goose up
go tool goose validate
```

Rules:
- Most tables: `uuid TEXT PRIMARY KEY` populated with `uuid.NewV7()`. Only internal tables may use `id INTEGER PRIMARY KEY`.
- UUID v7 is monotonically increasing (time-ordered), so usable as a cursor for keyset pagination (`WHERE uuid > ? ORDER BY uuid ASC LIMIT ?`).
- Track points (lat/lon/elevation in blobs) are immutable once inserted.
- DO NOT explicitly annotate cols with `NULL` if nullable - sqlc emits `interface{}` (NULL is implicit when omitted).
- Always store timestamps with timezone.
- Simple internal enums managed entirely in SQL: text + CHECK, e.g. `status TEXT CHECK(status IN ('C','R','A','E','S')) NOT NULL DEFAULT 'C'`.
- "App" enums managed from Go: integers declared with iota.
- Most tables have `created_at`, `updated_at`, `created_by`.
- IMPORTANT: when adding/changing columns on existing tables, ASK the operator whether to edit an existing migration or create a new one.
- IMPORTANT: when adding new columns to `users` or `email_verifications`, check whether they need to be in the demo-mode trigger lockdown in `internal/pkg/app/demo.go` (`demo_users_no_update` trigger's WHEN clause).

### Connection model

Each DB opens two connections (`internal/pkg/db/open.go`):
- `rw`: read/write, max 1 conn.
- `ro`: read-only, connection pool.

WAL mode, foreign keys on, busy timeout 5s. Per-DB pools mean independent write locks.

### Transactions

```
d.WithTx(ctx, func(tx *db.Queries) error { ... })  // auto commit/rollback
tx, err := d.BeginTX(ctx)                          // manual
```

### Transactions in REST handlers

Use `db.WithTx()` whenever a handler does a guard check then a mutation, to prevent TOCTOU.

Use sentinel errors to distinguish business-logic failures (4xx) from unexpected errors inside `WithTx`:

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
if errors.Is(err, errSomething)  { writeError(w, 409, "..."); return }
if err != nil                    { logg.Error(...); writeError(w, 500, "..."); return }
```

Note: `session.Create` and `session.Delete` open their own `WithTx` and cannot be nested. Call them before or after the handler's tx.

### Queries

- No ORM. SQL in `internal/pkg/db/queries/*.sql` (main), `internal/pkg/db/geonamesdb/queries/*.sql`, `internal/pkg/db/forecastdb/queries/*.sql`. Compiled to Go via sqlc. Run `make gen` after edits.
- When querying datetimes, you MUST wrap both sides with sqlite `datetime()` to handle timezones.
- Almost NEVER join in Go - use SQL JOINs and let SQLite do the work.
