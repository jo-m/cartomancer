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
- [ ] Middleware to insert auth/session/user info into request context
- [ ] Requests logging middleware
- [ ] Context logger insertion middleware
- [ ] https://phiresky.github.io/blog/2020/sqlite-performance-tuning/
- [ ] https://developer.android.com/topic/performance/sqlite-performance-best-practices
- [ ] Logging for goose
- [ ] Unified error handling and rendering
- [ ] Flag parsing, config, env vars
- [ ] Clean up endpoints
- [ ] Check in the generated files
- [x] Common API error response struct
- [x] Ensure fks are enforced in db
- [ ] Code TODOs
- [ ] XSRF protection
- [ ] Proper session handling -> https://github.com/alexedwards/scs
- [ ] Auto-restart
- [ ] Rate limiting for sensitive endpoints
- [ ] Better panic()s

# Hints
- Do not annotate cols with NULL, otherwise sqlc will emit interface{} (NULL is implicit anyways if left out)
- goose because: can do both sql and go

# Various
- https://betterstack.com/community/guides/logging/golang-contextual-logging/#using-context-context-with-slog
