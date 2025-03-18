# Commands

```bash
go get -tool github.com/pressly/goose/v3/cmd/goose
go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc
go get -tool github.com/valyala/quicktemplate/qtc

make check

go tool goose status
go tool goose create REPLACEME sql
go tool goose up
go tool goose validate

go tool sqlc generate
go tool sqlc vet

go tool air
```

# TODOs

- [x] Jobs: Delay, exponential backoff
- [ ] Email job
- [ ] Tests for job delay and backoff
- [ ] Forms/templates
- [ ] Unified error handling and rendering
- [ ] Clean up endpoints
- [ ] Code TODOs
- [ ] XSRF protection
- [ ] Clean up/refactor svc package
- [ ] CRUD users, self-registration
- [ ] Templates and static files

# Later TODOs

- [ ] Check in the generated files
- [ ] Rate limiting for sensitive endpoints
- [ ] TOTP Login
- [ ] VACUUM

# Hints
- Do not annotate cols with NULL, otherwise sqlc will emit interface{} (NULL is implicit anyways if left out)
- Goose because: can do both SQL and Go migrations
- We always try to store as little as possible in the database, and delete it right away. For investigations and debugging, we keep the logs instead.

# Various
- https://betterstack.com/community/guides/logging/golang-contextual-logging/#using-context-context-with-slog
