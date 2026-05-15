## REST API

### Middleware stack (applied in order in main.go)

- `chi.middleware.RequestID`
- `chi.middleware.ThrottleBacklog` - limits concurrent requests (if configured)
- `logg.AttachLogger` - attaches logger with request ID to context
- `logg.RequestLogger` - logs each request with duration/status
- `chi.middleware.RequestSize(5MB)`
- `chi.middleware.Compress(5)`
- `sess.Middleware` - auto-creates/loads session for every request
- `chi.middleware.Recoverer`

Per-route: `rateLimitByIP(rps, trustedProxies, ipv6PrefixLen, maxEntries)` (api.go) applies a per-IP token-bucket limit (429 on overflow, no-op if rps<=0) to `/sessions/login`, `/confirm-email`, and `/register`. IPv6 addresses are grouped by prefix. New IPs beyond `maxEntries` are allowed through (fail-open). Client IP is extracted from `X-Forwarded-For` when `trustedProxies > 0`.

### Context access in handlers

```
logg.Debug(ctx, "msg", "key", val)        // logger from context
logg.Error(ctx, "msg", "err", err)
session.MustGet(ctx)                      // current session (always set)
session.GetUser(ctx)                      // *db.User, nil if anonymous
```
