# Plan: Map view for filtered tracks

Add a toggle to the tracks page that switches between the existing list
view and a new map view rendering all currently-filtered tracks as
polylines on a single OpenLayers map.

## 1. UX

### View toggle
- `<ToggleGroup>` (already in `src/components/ui/`) labelled "List / Map"
  with list/map heroicons, anchored at the top-right of the filter bar.
- View persisted in URL via `useUrlState` (`view=list|map`,
  `enumParam`). Default `list` (preserves current link semantics).
- Filter panel (`TrackFilters`) is shared between both views and stays
  visible above either canvas. URL filter state wins; both views
  re-render on filter change.

### Map view layout
- Map fills the page below filters at `h-[calc(100vh-Xrem)]`. On mobile,
  full viewport minus header.
- Floating overlay (top-right of the map): count of rendered tracks, a
  "fit all" button, layer attribution.
- Empty state: "No tracks match. Adjust filters." centered on a blank
  map.
- Cap: **250 tracks** rendered. If the filtered set is larger, show a
  banner "Showing 250 of M; refine filters to see all."

### Hover & click
- Hovering a polyline highlights it and shows a small popover with track
  name, distance, ascent, and a `<SvgPreview>` thumbnail
  (`/api/tracks/{uuid}/preview.svg?size=128`).
- The popover is wrapped in a real `<Link to="/tracks/<uuid>">` so
  middle-click and right-click work (per `frontend/CLAUDE.md`).
- If multiple features are at the cursor pixel, the popover lists them.

### Selection mode
- When `BulkEditToolbar` is active, clicking a polyline toggles its
  selection (parity with `TrackCard`). Selected tracks render with a
  thicker primary-coloured halo.
- Selection state lifts from `TrackGrid` into a parent that hosts both
  views, so toggling view preserves selection.

### Viewport persistence
- After pan/zoom, persist `lat,lon,zoom` in the URL (e.g. `m=lat,lon,z`,
  debounced 300ms) so reloads keep position.
- Initial fit: union of all rendered track bboxes,
  `view.fit(unionExtent, { padding, maxZoom: 11 })`.

## 2. Backend

The blocker is the polyline source. `/tracks/{uuid}/points` loads each
track's blob (zstd-compressed GPX/FIT, up to 5MiB), parses it, and runs
LTTB. Doing this for 250 tracks per page load is unacceptable. Solution:
store a low-resolution polyline once at upload time.

### 2.1 Simplifier
- New helper `(Points).SimplifyDP(epsilonM float64) Points` in
  `internal/pkg/track/`.
- Algorithm: Douglas–Peucker, with perpendicular distance computed in
  metres on the WGS84 ellipsoid (or close-enough equirectangular
  projection at the local latitude — the existing `MetersTo` is fine).
- Tolerance const: `PreviewPolylineEpsilonM = 200.0` in
  `internal/pkg/track/`. Adaptive density: straight stretches collapse
  to 2 points; switchbacks keep many.
- Existing `SubsampleLTTB` is 1-dimensional (`valueFn(Point) float64`,
  cumulative distance as X) — useful for the elevation sparkline, not
  for 2D polyline simplification. Not reused.
- Tests: golden-file snapshot tests on representative tracks.

### 2.2 Encoded polyline
- New helpers `EncodePolyline(pts Points) string` and `DecodePolyline(s
  string) Points` in `internal/pkg/track/` (Google polyline algorithm,
  precision 5 — ~1.1m at the equator, plenty for a 200m-tolerance
  simplification).
- Output is ASCII (printable chars 63–126).

### 2.3 Schema
- New migration `internal/pkg/db/migrations/00000000000018_track_preview_polyline.sql`:
  ```sql
  ALTER TABLE tracks ADD COLUMN preview_polyline TEXT;
  ```
- `TEXT` (not `BLOB`): encoded polyline is ASCII; `TEXT` is debuggable
  in the `sqlite3` REPL and round-trips cleanly through sqlc as `string`
  rather than `[]byte`. Storage cost on disk is identical.
- Nullable until backfill completes.

### 2.4 Compute
- Wire computation into `handleUploadTrack` (`internal/pkg/api/track.go`) so
  every new track gets a polyline at insert time. The track is already
  parsed and bounds are computed there; piggy-back on that.
- Backfill via a job in `internal/pkg/jobs/`:
  - Args: `BackfillPreviewPolylineArgs{}` (no parameters; processes one
    track per run, reschedules itself if more remain).
  - Selects `uuid FROM tracks WHERE preview_polyline IS NULL ORDER BY
    uuid LIMIT 1`, computes, updates, schedules next.
  - One-shot enqueue at startup if any rows are NULL.

### 2.5 New endpoint
```
GET /api/tracks/polylines?<same filters as /tracks>&limit=250
→ {
    tracks: [{
      uuid,
      name,
      totalDistanceM,
      totalAscentM,
      polyline,                 // encoded polyline string
      bounds: { minLat, minLon, maxLat, maxLon },
      user: { uuid, name },
      starred?,
      isOwner?,
    }],
    totalCount,
    truncated: boolean,         // true when totalCount > limit
  }
```
- Implementation in `internal/pkg/api/track.go` as
  `handleListTrackPolylines`.
- Reuses the same filter parsing as `handleListTracks` — extract the
  query-param parsing block into a shared helper
  `parseListTracksParams(r) (db.ListTracksParams, error)` so filters
  stay consistent across both endpoints.
- Hard cap: `limit ≤ 1000`. Default 250.
- `Cache-Control: private, max-age=60`. `ETag` based on
  `MAX(updated_at)` over the matched set + a hash of the filter
  parameters. `If-None-Match` returns `304`.
- Tracks with `preview_polyline IS NULL` are skipped in the response
  (so the frontend can render whatever is ready while backfill
  proceeds). Response includes `pendingCount` so the UI can show an
  "indexing tracks…" indicator.
- Compress middleware already covers the response.

### 2.6 sqlc
- New queries in `internal/pkg/db/queries/tracks.sql`:
  - `SetTrackPreviewPolyline` — `UPDATE tracks SET preview_polyline = ?
    WHERE uuid = ?`.
  - `ListTracksWithPolylines` — same predicate as `ListTracks`, joined
    with whichever supplementary columns the polylines endpoint
    returns; includes `preview_polyline`.
  - `CountTracksMissingPreviewPolyline` — for backfill scheduling.
- Run `make gen` after editing.

### 2.7 OpenAPI
- Add `/tracks/polylines` path and `TrackPolylinesResponse` /
  `TrackPolylineEntry` schemas to
  `internal/pkg/api/openapi.yaml`.
- Frontend types regenerate via `npm run gen` (already in `dev`/`build`).

## 3. Frontend

### 3.1 New component
- `frontend/src/components/TracksMapView.tsx`.
- Props: filter params (same shape as `TrackGrid` passes to
  `useQuery("get", "/tracks")`), selection state, callbacks.
- Fetches `$api.useQuery("get", "/tracks/polylines", { params: { query
  } })`.

### 3.2 Polyline decode
- New `frontend/src/lib/polyline.ts`: ~30-line Google polyline decoder
  (precision 5). Avoid pulling a dep.
- Tests via `vitest` if/when tests land; until then a small in-file
  sanity check vector covered by a manual run.

### 3.3 OpenLayers integration
- One `VectorSource` with one `Feature(LineString)` per track. Single
  `VectorLayer` for all of them — never one layer per feature
  (compositing cost dominates above ~50 layers).
- Reproject lat/lon → map projection using the same `projectPoint(lon,
  lat, layer.type)` helper currently in `TrackMap.tsx` (lift into
  `frontend/src/lib/proj.ts` so both components share it).
- Feature properties: `uuid`, `name`, `totalDistanceM`, `totalAscentM`,
  `selected`, `hovered`.
- Layer-level style function reads those properties:
  - 4px coloured stroke (per-track colour from `lib/trackColor.ts` for
    visual distinction) with a 7px white halo, matching `TrackMap`.
  - Selected: thicker primary-coloured halo.
  - Hovered: brighter / slightly thicker.
  - Below zoom 8: drop the halo, drop to 2px stroke (compositing is the
    bottleneck at world-scale views).
- Hover: `forEachFeatureAtPixel` with `hitTolerance: 6`, throttled via
  `requestAnimationFrame`.

### 3.4 Layer selection
- `selectMapLayer` in `frontend/src/lib/mapLayer.ts` operates on one
  bbox today. Add `unionBbox(bboxes: Bbox[]): Bbox | null` and feed
  the union into `selectMapLayer` so SwissTopo is used iff every
  rendered track is inside the SwissTopo coverage area; otherwise the
  global PMTiles build (or `none`) wins.

### 3.5 TrackGrid changes
- Lift `view`, `selected`, and `lastClickedIndex` into a small parent
  (or keep them in `TrackGrid` and conditionally render either the
  card grid or `<TracksMapView>`).
- Add the `<ToggleGroup>` toggle next to the existing pagination
  controls (visible only after the data has loaded so layout doesn't
  flicker on first paint).
- Pagination controls hide in map mode.

### 3.6 Code-split
- Dynamic-import `TracksMapView` (and via it OpenLayers + ol-pmtiles)
  so list-only users don't pay the bundle cost on the tracks page.
  OpenLayers + `ol-pmtiles` is several hundred KiB. The Track detail
  page already pulls them in, but the tracks listing page should not
  by default.

### 3.7 Lint / CSP
- After the change, `npm run lint` must pass.
- No new external origins are introduced (PMTiles is already
  self-hosted at `/api/maps/{uuid}`), so the CSP plugin in
  `vite.config.ts` does not need updates.

## 4. Phase 2 (deferred)

Not in this change, but kept in mind so the API doesn't paint us into
a corner:

- "Limit to map area" toggle that adds a bbox-intersects filter and
  re-queries on `moveend` (debounced 400ms). Today's `ListTracksParams`
  filters by start/end point only; we'd add a `bboxIntersects` filter
  using the existing `bounds_min_*` / `bounds_max_*` columns.
- Useful for users with many thousands of tracks.

## 5. Files

### Backend
- `internal/pkg/db/migrations/00000000000018_track_preview_polyline.sql` — new.
- `internal/pkg/track/simplify.go` — DP simplifier + const + tests.
- `internal/pkg/track/polyline.go` — encode/decode helpers + tests.
- `internal/pkg/db/queries/tracks.sql` — new queries.
- `internal/pkg/api/track.go` — `handleListTrackPolylines`; refactor
  filter parsing into a helper used by both endpoints.
- `internal/pkg/api/api.go` — register route `GET /tracks/polylines`.
- `internal/pkg/api/openapi.yaml` — new path + schemas.
- `internal/pkg/jobs/backfill_preview_polyline.go` — backfill job +
  registration in `main.go`.
- `internal/pkg/api/track.go` — call simplifier + encoder in
  `handleUploadTrack`.

### Frontend
- `frontend/src/components/TracksMapView.tsx` — new.
- `frontend/src/components/TrackGrid.tsx` — view toggle, conditional
  render.
- `frontend/src/lib/polyline.ts` — decoder.
- `frontend/src/lib/mapLayer.ts` — `unionBbox`.
- `frontend/src/lib/proj.ts` — lift `projectPoint` here from
  `TrackMap.tsx` and reuse.

## 6. Performance recap

- Per request to `/tracks/polylines`: one indexed `SELECT` over
  `tracks` returning ASCII polylines. No blob reads, no GPX/FIT parsing,
  no LTTB.
- Payload: encoded polyline ~3-5 bytes/point. A 50km track at
  epsilon=200m typically yields ~50–150 points → ~250–750 bytes per
  track. 250 tracks ≈ 70-200 KiB on the wire (gzipped ~30-80 KiB).
- Frontend renders all 250 tracks in a single `VectorLayer`, single
  `VectorSource`, layer-level style fn. No per-feature `setStyle`
  except on hovered/selected.
- Code-split keeps OL out of the listing-page bundle for list-only
  users.

## 7. Decisions baked in

- Algorithm: Douglas–Peucker.
- Tolerance: `PreviewPolylineEpsilonM = 200.0` (configurable via the
  const).
- Storage: `preview_polyline TEXT` column on `tracks`, new migration.
- Default cap: 250 tracks per map view.
- Backfill: async job, idempotent, one-shot enqueue at startup.
