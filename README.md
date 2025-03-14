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

- [x] https://phiresky.github.io/blog/2020/sqlite-performance-tuning/
- [ ] https://developer.android.com/topic/performance/sqlite-performance-best-practices
- [ ] Unified error handling and rendering
- [ ] Clean up endpoints
- [ ] Code TODOs
- [ ] XSRF protection
- [x] Auto-restart https://github.com/air-verse/air, https://eradman.com/entrproject/, https://github.com/cortesi/modd
- [ ] Clean up/refactor svc package
- [ ] CRUD users, self-registration
- [x] Session data
- [x] Ensure non-threadsafe SQLite sessions are not accidentally shared.
- [x] e2e tests for: Login, logout, session expiry
- [x] Wrap session actions in db tx
- [ ] Templates and static files
- [x] Job queue
- [x] Scheduled jobs
- [ ] Make almost all queries `:execrows`
- [ ] Test and document and use InTx()
- [ ] Set min age for jobs cleanup
- [x] Change DB locking by adding to each job the PID
- [ ] Exponential backoff

# Later TODOs

- [ ] Check in the generated files
- [ ] Rate limiting for sensitive endpoints
- [ ] TOTP Login
- [ ] Clean up sessions periodically
- [ ] VACUUM

# Hints
- Do not annotate cols with NULL, otherwise sqlc will emit interface{} (NULL is implicit anyways if left out)
- Goose because: can do both SQL and Go migrations
- We always try to store as little as possible in the database, and delete it right away. For investigations and debugging, we keep the logs instead.

# Various
- https://betterstack.com/community/guides/logging/golang-contextual-logging/#using-context-context-with-slog
