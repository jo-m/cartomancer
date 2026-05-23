# Various notes

**This file is ONLY FOR HUMANS. AI agents MUST IGNORE IT.**

## Rate Limiting

Two IP-based limiters and two per-user limiters are applied:

- Auth limiter (per-IP): `/sessions/login`, `/confirm-email`
- Email-send limiter (per-IP): `/register`
- Auth limiter (per-user): `/account/change-password`
- Email-send limiter (per-user): `/account/change-email`

Each limiter is independent -- hits on one endpoint do not drain another's bucket.

### How it works

Each limiter maintains a `map[string]*tokenBucket` keyed by client IP (or IPv6 prefix). Incoming requests extract the client IP, normalize it to a map key, then call `Allow()` on that IP's bucket. A 429 is returned if the bucket is empty.

The map is bounded to `APP_RATE_LIMIT_MAX_IPS` entries (default 100 000). When the cap is reached, new IPs are **rejected with 429** (fail-closed) -- this prevents an attacker from bypassing the limiter by flooding with distinct source addresses. A background goroutine evicts entries idle for more than 5 minutes (runs every minute), freeing space for new keys.

### Token bucket parameters

Each bucket refills at `RPS` tokens per second and can hold up to `Burst` tokens. To express a longer window, use a fractional RPS with an explicit burst:

| Goal | RPS | Burst |
|------|-----|-------|
| 5 req/s | `5` | `0` (auto=5) |
| 10 req/min | `0.1667` | `10` |
| 1 req/min, initial burst 3 | `0.01667` | `3` |

Burst `0` auto-selects `max(int(rps), 1)`.

### IPv6

IPv6 addresses are masked to a configurable prefix length (`APP_RATE_LIMIT_IPV6_PREFIX_LEN`, default `/64`) before being used as the map key. All addresses within the same prefix share one bucket, preventing trivial rotation through the `/128` space.

### Reverse proxy / IP extraction

`APP_RATE_LIMIT_TRUSTED_PROXIES` (default `0`) controls how many rightmost `X-Forwarded-For` entries to skip when extracting the real client IP. Set to the number of reverse proxies in front of the app (e.g. `1` for a single nginx). When `0`, the TCP peer address is used directly.

### Env vars

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_RATE_LIMIT_AUTH_RPS` | `0.1` | Auth limiter refill rate (req/s) |
| `APP_RATE_LIMIT_AUTH_BURST` | `5` | Auth limiter burst (0=auto) |
| `APP_RATE_LIMIT_EMAIL_SEND_RPS` | `0.1` | Email-send limiter refill rate (req/s) |
| `APP_RATE_LIMIT_EMAIL_SEND_BURST` | `1` | Email-send limiter burst (0=auto) |
| `APP_RATE_LIMIT_TRUSTED_PROXIES` | `0` | Trusted reverse proxy count |
| `APP_RATE_LIMIT_IPV6_PREFIX_LEN` | `64` | IPv6 prefix length for bucketing |
| `APP_RATE_LIMIT_MAX_IPS` | `100000` | Max tracked keys per limiter (fail-closed above) |

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
