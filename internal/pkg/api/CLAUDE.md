## Rest API

### Middleware stack (applied in order in main.go)

- chi.middleware.RequestID
- chi.middleware.ThrottleBacklog: limits concurrent requests (if configured)
- logg.AttachLogger: attaches logger with request ID to context
- logg.RequestLogger: logs each request with duration/status
- chi.middleware.RequestSize(5MB)
- chi.middleware.Compress(5)
- sess.Middleware: auto-creates/loads session for every request
- chi.middleware.Recoverer

### Context access in handlers

```
// logger from context
logg.Debug(ctx, "msg", "key", val)
logg.Error(ctx, "msg", "err", err)

// current session (always set)
session.MustGet(ctx)
// *db.User, nil if anonymous
session.GetUser(ctx)
```
