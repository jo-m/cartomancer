# Commands

```bash
go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
go get -tool github.com/pressly/goose/v3/cmd/goose
go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc
go get -tool github.com/valyala/quicktemplate/qtc
go get -tool github.com/mailhog/MailHog

make check

go tool goose status
go tool goose create REPLACEME sql
go tool goose up
go tool goose validate

go tool sqlc generate
go tool sqlc vet

go tool air
```

# Email

```bash
# http://127.0.0.1:8025/
go tool MailHog
```

# TODOs

- [x] Jobs: Delay, exponential backoff
- [x] Make BackofFactorS configurable per Job
- [x] Tests for job delay and backoff
- [x] Email job
- [ ] Allow submitting jobs in a transaction
- [ ] Clean up jobs with min age
- [ ] Frameworkeize/factor out was much as possible
- [ ] Maybe keep session ID in signed JWT
- [ ] Unified error handling and rendering
- [ ] Clean up endpoints
- [ ] Code TODOs
- [ ] TS Oapi generator: https://www.npmjs.com/package/openapi-typescript
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
- JSON API endpoints should always require the `Accept` header to ensure errors are rendered correctly.

# Various

App ideas:
- Shopping list
- Dashboard/feed reader
- Bookmarks
