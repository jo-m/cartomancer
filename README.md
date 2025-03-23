# Commands

```bash
go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
go get -tool github.com/pressly/goose/v3/cmd/goose
go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc
go get -tool github.com/mailhog/MailHog
go get -tool github.com/a-h/templ/cmd/templ

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
- [x] Maybe use https://github.com/a-h/templ instead
- [x] Consider https://github.com/a-h/rest --> No.
- [x] Remove qtpl
- [ ] Get URLs from OpenAPI spec in templates
- [x] Static files
- [ ] Separate endpoints into HTTP + Service layer
- [x] Make auto-restart work with templ
- [x] Allow submitting jobs in a transaction
- [ ] Maybe keep session ID in signed JWT
- [x] Unified error handling
- [x] Clean up endpoints
- [ ] Code TODOs
- [ ] XSRF protection
- [ ] CRUD users, self-registration
- [ ] Try https://hotwired.dev/ and HTMX
- [x] Jobs: Cleanup with min age
- [x] Jobs: execution timeouts

# Later TODOs

- [ ] Frameworkeize/factor out as much as possible
- [ ] TS Oapi generator: https://www.npmjs.com/package/openapi-typescript
- [ ] Check in the generated files
- [ ] Rate limiting for sensitive endpoints
- [ ] (pre)compress static files
- [ ] TOTP Login
- [ ] VACUUM

# Hints
- Do not annotate cols with NULL, otherwise sqlc will emit interface{} (NULL is implicit anyways if left out)
- Goose because: can do both SQL and Go migrations
- We always try to store as little as possible in the database, and delete it right away. For investigations and debugging, we keep the logs instead.
- Server handlers should never return (nil, err)

# Various

App ideas:
- Shopping list
- Dashboard/feed reader
- Bookmarks
