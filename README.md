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
- [x] Middleware to insert auth/session/user info into request context
- [x] Requests logging middleware
- [x] Context logger insertion middleware
- [ ] https://phiresky.github.io/blog/2020/sqlite-performance-tuning/
- [ ] https://developer.android.com/topic/performance/sqlite-performance-best-practices
- [x] Logging for goose
- [ ] Unified error handling and rendering
- [ ] Flag parsing, config, env vars
- [ ] Clean up endpoints
- [ ] Check in the generated files
- [x] Common API error response struct
- [x] Ensure fks are enforced in db
- [ ] Code TODOs
- [ ] XSRF protection
- [x] Proper session handling -> https://github.com/alexedwards/scs
- [ ] Auto-restart
- [ ] Rate limiting for sensitive endpoints
- [x] Better panic()s
- [ ] Clean up sessions periodically
- [ ] Track user last login/active
- [x] Make user and session IDs be UUIDs
- [ ] TOTP Login
- [ ] Clean up/refactor svc package
- [ ] Maybe store the uuids in the db as bytes
- [ ] CRUD users, self-registration

# Hints
- Do not annotate cols with NULL, otherwise sqlc will emit interface{} (NULL is implicit anyways if left out)
- goose because: can do both sql and go

# Various
- https://betterstack.com/community/guides/logging/golang-contextual-logging/#using-context-context-with-slog
