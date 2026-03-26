# CLAUDE.md

This is a web app to manage GPX tracks for cycling/running.
Backend is a Golang REST API.
Frontend is React + TypeScript + Vite in `frontend/`.

```bash
# Run backend, implicitly configured by `.envrc`.
go run .
```

# Backend

## Package structure

```
internal/pkg/
├── app/      # Application-level config
├── blob/     # Blob storage for GPX/FIT files (zstd-compressed, SQLite)
├── db/       # SQLite connection, migrations, sqlc-generated queries
├── jobs/     # Persistent async job queue
├── load/     # GPX/FIT file parsing → TrackSource
├── logg/     # Structured logging (slog), middleware, context helpers
├── mail/     # Email job handler (SMTP via go-mail)
├── password/ # Argon2id hashing
├── rest/     # HTTP handlers (Chi router)
├── session/  # JWT+cookie session management, middleware
├── track/    # Track types, enums, metadata calculations
├── users/    # OTP (TOTP/HOTP) support
└── utl/      # General utilities
```

### REST API

Handlers are mounted in `internal/pkg/api/rest.go`.
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
2. `logg.AttachLogger` — attaches logger with request ID to context
3. `logg.RequestLogger` — logs each request with duration/status
4. `chi.middleware.RequestSize(1MB)`
5. `chi.middleware.Compress(5)`
6. `chi.middleware.RedirectSlashes`
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

### Migrations

github.com/pressly/goose/v3 and are placed in `internal/pkg/db/migrations/*.sql`.
To create a new one:
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

Two connections are opened (`internal/pkg/db/open.go`):
- `rw` — read/write, max 1 connection (SQLite requirement)
- `ro` — read-only, connection pool

WAL mode, foreign keys enabled, busy timeout 5s.

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
Written in SQL in `internal/pkg/db/queries/*.sql`.
Compiled to Go methods via sqlc.
Run `make gen` after editing queries.

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

Use `db.GetTestDB(t)` to get a temp SQLite DB with all migrations applied.
Use `github.com/stretchr/testify/require` for assertions.
Use `https://github.com/franiglesias/golden` for snapshot tests. Approval mode: `golden.Verify(t, output, golden.WaitApproval())`. Set custom extension for snapshot files: golden.Verify(t, output, golden.Extension(".json")).

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

# Frontend

## Directory structure

```
frontend/
├── src/
│   ├── api/
│   │   ├── client.ts    # openapi-fetch + openapi-react-query setup
│   │   └── schema.gen.ts  # Generated by openapi-typescript (do not edit)
│   ├── pages/           # Page components (one per route)
│   ├── App.tsx           # Router + layout shell
│   ├── main.tsx          # Entry point
│   └── index.css         # Tailwind imports
├── index.html
├── vite.config.ts
├── eslint.config.js
├── .prettierrc
├── tsconfig.json
└── package.json
```

## Development

```bash
cd frontend
npm run dev
npm run check
npm run build
```

## API client

Types are generated from `internal/pkg/api/openapi.yaml` via `openapi-typescript`.
Run `npm run generate` (already included in `dev`/`build`) to regenerate `src/api/schema.gen.ts`.
`src/api/schema.gen.ts` is NOT committed.
All API interactions MUST use the generated client.

`src/api/client.ts` exports:
- `fetchClient` — `openapi-fetch` client with base URL `/api`. Has a middleware that converts API error bodies `{ msg }` into thrown `Error` instances. Use for direct calls (e.g. in `SessionContext`).
- `$api` — `openapi-react-query` wrapper around `fetchClient`. Use in components.
- `User` — convenience type re-export.

### Data fetching in components

Use `$api.useQuery` for reads and `$api.useMutation` for writes:

```ts
const { data, isLoading, error } = $api.useQuery("get", "/some-resource")
const mutation = $api.useMutation("post", "/some-resource")
// mutation.isPending, mutation.isSuccess, mutation.error
```

Errors from mutations/queries are `Error` instances at runtime (middleware converts API errors).
The TypeScript-inferred error type may be `{ msg: string }` (from OpenAPI schema) — cast as needed: `(mutation.error as unknown as Error).message`.

### Forms

Use `react-hook-form` with `zod` schemas for all forms:

```ts
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"

const schema = z.object({ name: z.string().min(1, "Required") })
const { register, handleSubmit, formState: { errors, isSubmitting }, setError } = useForm({
  resolver: zodResolver(schema),
  values: { name: existingValue }, // use `values` (not defaultValues) to sync with external state
})

async function onSubmit(data) {
  try {
    await mutation.mutateAsync({ body: data })
  } catch { /* mutation.error is set */ }
}
```

Use `setError("root", { message: ... })` for server-side errors on login/similar flows.
Use `mutation.isSuccess` / `mutation.error` for success/error display on mutation forms.

## Styling

Tailwind CSS v4 with `@tailwindcss/vite` plugin. CSS-based config (no `tailwind.config.js`).
Use Tailwind utility classes directly. `@headlessui/react` for accessible interactive components, `@heroicons/react` for icons.
Keep it very simple and barebones.

## Conventions

- Vite build outputs to `../static/` (embedded in Go binary from there).
- All data views/tables must always be searchable/paginatable/filterable.
- All links, including nav etc. must be proper `<a>` links such that right click, open in new tab etc. work as expected.
- URL paths used in the router should generally roughly mirror those from the API. E.g. the tracks upload page (POST /api/tracks) should be at /tracks/uploads.
- Error handling: Unless API errors have some separate handling mechanism like e.g. the failed uploads list on the upload page, use the toast element (frontend/src/components/Toast.tsx) to forward API errors to the user.
- External links: Any `href` sourced from the API or database (e.g. track author links, attribution URLs) must NOT link directly to the external site. Instead, route them through the `/leaving` interstitial page using the `externalUrl()`. Links hardcoded in the backend are exempt.
- NEVER must any assets in the frontend be loaded from a third party domain. All assets must be included in the build.

## Linting

After every change, `npm run check` MUST run successfully.

# Universal Guidelines

- NEVER print or log URLs to console if they contain an API key
- NEVER hardcode sensitive configuration (keys/passwords) into the code
- MUST keep functions focused on a single responsibility
- MUST include docstrings for all public functions, classes, and methods
  - MUST document function parameters, return values, and exceptions raised
  - Keep comments up-to-date with code changes
- MUST use meaningful, descriptive variable and function names
- NEVER use emoji, or unicode that emulates emoji in the source code (e.g. ✓, ✗). The only exception is when writing tests and testing the impact of multibyte characters.  
- MUST avoid including redundant comments which are tautological or self-demonstrating (e.g. cases where it is easily parsable what the code does at a glance so the comment does)
- MUST avoid including comments which leak what this file contains, or leak the original user prompt, ESPECIALLY if it's irrelevant to the output code.
