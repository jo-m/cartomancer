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

Per-route: `rateLimit(rps)` (api.go) applies a global token-bucket limit (429 on overflow, no-op if rps<=0) to `/sessions/login`, `/confirm-email`, and `/register`.

### Context access in handlers

```
logg.Debug(ctx, "msg", "key", val)        // logger from context
logg.Error(ctx, "msg", "err", err)
session.MustGet(ctx)                      // current session (always set)
session.GetUser(ctx)                      // *db.User, nil if anonymous
```
