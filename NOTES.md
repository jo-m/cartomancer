# Various notes

**This file is ONLY FOR HUMANS. AI agents MUST IGNORE IT.**

## ETag Usage

### How it works today

The backend sets `ETag` response headers and checks `If-None-Match` on all
cacheable endpoints (track points, polylines, SVG previews, avatar). When the
header matches, the server returns 304 Not Modified.

The frontend does not handle ETags explicitly. The browser's HTTP cache
intercepts 304 responses and serves the cached body transparently, so
`openapi-fetch` and React Query never see a 304.

### Problem: max-age conflicts with React Query invalidation

JSON API endpoints use `Cache-Control: private, max-age=X`:

- `/tracks/{uuid}/points` -- `max-age=3600`
- `/tracks/polylines/5m` and `/tracks/polylines/50m` -- `max-age=60`

React Query's global `staleTime` is 2 minutes. When a mutation invalidates a
query and React Query triggers a refetch, the browser HTTP cache may serve a
stale cached response directly if the entry is still within `max-age`. The
ETag conditional GET never fires and React Query silently receives stale data.

### Fix: use `no-cache` on JSON endpoints

Replace `max-age=X` with `no-cache` on the JSON API endpoints above.
`Cache-Control: private, no-cache` means the browser always validates with the
server before using a cached copy, so `If-None-Match` is sent on every
request. The server still returns 304 when nothing changed (saving bandwidth),
but React Query's explicit invalidations now always reach the server.

### SVG endpoints are fine as-is

`/tracks/{uuid}/preview.svg`, `/tracks/{uuid}/profile.svg`, and
`/users/{uuid}/avatar` are loaded via `<img>` tags outside React Query's
control. Their `max-age` caching with ETag validation is the correct pattern
for these resources.

## Rate-limit candidates

Unauthenticated backend endpoints that need rate limiting because they enable
account brute-forcing or outbound email abuse. Routes are mounted in
`internal/pkg/api/api.go`. The current middleware stack only enforces
`chi.middleware.ThrottleBacklog` (global concurrency cap); there is no
per-route or per-actor rate limiting.

### Brute-force surfaces

These accept passwords or tokens that an attacker can guess. Limit by IP and,
where an identifier is supplied, per-target (email / token subject). Use slow,
low-burst limits.

| Method | Path | Handler | Why sensitive |
| --- | --- | --- | --- |
| POST | `/sessions/login` | `handleLogin` (`session.go:52`) | Password brute force against any known email. argon2id verification is CPU-expensive, so unbounded attempts are also a DoS lever. |
| POST | `/confirm-email` | `handleConfirmEmail` (`registration.go:193`) | Accepts a JWT verification token; rate limit to slow token-guessing and replay. |

### Email-sending surfaces

Each successful call submits a `mail.Args` job, sending real email through the
configured mailer. Without limits an attacker can spam arbitrary inboxes and
burn outbound mail quota. Limit per IP, per target email, and globally.

| Method | Path | Handler | Why sensitive |
| --- | --- | --- | --- |
| POST | `/register` | `handleRegister` (`registration.go:35`) | Sends a confirmation email (new email) OR a "someone tried to register" email (existing email) to any attacker-supplied address. Trivial inbox-flooding primitive. |

### Suggested grouping for implementation

Two buckets cover everything above:

1. **`auth`** (very strict, per-IP and per-target): `/sessions/login`,
   `/confirm-email`.
2. **`email-send`** (very strict, per-IP, per-target-email, and a global cap):
   `/register`.
