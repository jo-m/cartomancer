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
```

# TODOs

- [ ] https://phiresky.github.io/blog/2020/sqlite-performance-tuning/
- [ ] https://developer.android.com/topic/performance/sqlite-performance-best-practices
- [ ] Unified error handling and rendering
- [ ] Clean up endpoints
- [ ] Code TODOs
- [ ] XSRF protection
- [ ] Auto-restart
- [ ] Track user last login/active
- [ ] Clean up/refactor svc package
- [ ] CRUD users, self-registration
- [ ] Session data
- [ ] Ensure non-threadsafe SQLite sessions are not accidentally shared.
- [ ] e2e tests for: Login, logout, session expiry

# Later TODOs

- [ ] Check in the generated files
- [ ] Rate limiting for sensitive endpoints
- [ ] TOTP Login
- [ ] Clean up sessions periodically


# Hints
- Do not annotate cols with NULL, otherwise sqlc will emit interface{} (NULL is implicit anyways if left out)
- goose because: can do both sql and go

# Various
- https://betterstack.com/community/guides/logging/golang-contextual-logging/#using-context-context-with-slog
