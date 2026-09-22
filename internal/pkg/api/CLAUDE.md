## REST API

### Middleware stack (applied in order in main.go)

On all routes:

- `chi.middleware.RequestID`
- `chi.middleware.ThrottleBacklog` - limits concurrent requests (if configured)
- `logg.AttachLogger` - attaches logger with request ID to context
- `logg.RequestLogger` - logs each request with duration/status
- `chi.middleware.RequestSize(5MB)`
- `chi.middleware.Compress(5)`
- `chi.middleware.Recoverer` - recovers handler panics, reported through the
  context logger (see `logg.logEntry.Panic`)

On `/api` only. The SPA and `/robots.txt` are served without session context, since
they never read it and loading a session costs a database round-trip:

- `sess.Middleware` - loads the session of the request, if there is one

Per-route:
- `rateLimitByIP(rps, ...)` (api.go): per-IP token-bucket limit (429 on overflow, no-op if rps<=0) on `/sessions/login`, `/confirm-email`, and `/register`. IPv6 addresses are grouped by prefix. New IPs beyond `maxEntries` are rejected (fail-closed) until the cleanup goroutine evicts idle entries. Client IP from `X-Forwarded-For` when `trustedProxies > 0`.
- `rateLimitByUID(rps, burst, maxEntries)` (api.go): per-authenticated-user token-bucket limit on `/account/change-password` (auth RPS/burst) and `/account/change-email` (email-send RPS/burst). Falls through (fail-open) when no user is in context.

### Context access in handlers

```
logg.Debug(ctx, "msg", "key", val)        // logger from context
logg.Error(ctx, "msg", "err", err)
session.MustGet(ctx)                      // current session (panics if anonymous)
session.GetUser(ctx)                      // *db.User, nil if anonymous
```
