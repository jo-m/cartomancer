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
