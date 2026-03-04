# Commands

```bash
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
- [x] Static files
- [x] Make auto-restart work with templ
- [x] Allow submitting jobs in a transaction
- [x] Keep session ID in signed JWT
- [x] Unified error handling
- [x] Clean up endpoints
- [x] Get URLs from OpenAPI spec
- [x] Modify generated links to return safe URLs (but escape string params)
- [ ] Users self-registration, password reset, email confirm
- [ ] Code TODOs
- [ ] XSRF protection
- [x] Jobs: Cleanup with min age
- [x] Jobs: execution timeouts
- [x] Go doc links `[]` https://tip.golang.org/doc/comment, `go run golang.org/x/pkgsite/cmd/pkgsite@latest -open`
- [ ] Skills/subagents

# Later TODOs

- [ ] https://brandur.org/two-phase-render
- [ ] TS Oapi generator: https://www.npmjs.com/package/openapi-typescript
- [ ] Check in the generated files
- [ ] Rate limiting for sensitive endpoints
- [ ] (pre)compress static files
- [ ] VACUUM
