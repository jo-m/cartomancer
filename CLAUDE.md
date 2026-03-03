# CLAUDE.md

This is a web app to manage GPX tracks for cycling/running.
Backend is a Golang REST API.
The frontend does NOT exist yet.

```bash
# Run backend, implicitly configured by `.envrc`.
go run .
```

# REST API

Handlers are mounted in `internal/pkg/rest/rest.go`.
Group handler fns into files, approx. 1 per resource, in `internal/pkg/rest/*.go`.
They all must be methods on the `Server` struct.
`internal/pkg/rest/openapi.yaml` MUST be updated when ever endpoints change.
Follow RESTful API design guidelines, and use appropriate HTTP methods and status codes.
Use camelCase for any JSON fields (e.g. "SessionID string `json:"sessionId"`").
Use the helpers in `internal/pkg/rest/error.go`.
In most cases where an error is returned from a handler, the details should be logged.

# Database

## Migrations

github.com/pressly/goose/v3 and are placed in `internal/pkg/db/migrations/*.sql`.
To create a new one:
```bash
go tool goose status
go tool goose create REPLACEME sql
go tool goose up
go tool goose validate
```
DO NOT annotate cols with NULL, otherwise sqlc will emit interface{} (NULL is implicit anyways if left out).
Always store timestamps with timezone.
Enum are strings with CHECK, example: `status TEXT CHECK(status IN ('C', 'R', 'A', 'E', 'S') ) NOT NULL DEFAULT 'C'`.

## Queries

No ORM.
Written in SQL in `internal/pkg/db/queries/*.sql`.
Compiled to Go methods via sqlc.
Run `make gen` after editing queries.

# Linting and code quality

After every change, `make check` MUST run successfully.

# Generic

- All files with ending `.gen.go` are generated and MUST NOT EVER be edited manually.
- Usually the logger instance is passed around in ctx.Context
- Modules have config structs if applicable, compatible with github.com/alexflint/go-arg, example `internal/pkg/logg/handler.go`.
