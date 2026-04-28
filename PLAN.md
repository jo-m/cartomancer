# Plan: replace `SubsampleLTTB()` with DP + min-distance subsampling

## 1. Where `SubsampleLTTB()` is used today

### 1.1 Backend call sites

All four production call sites use the same `valueFn`: `p.Elevation`.

| File | Function | Target N | Consumer |
| --- | --- | --- | --- |
| `internal/pkg/api/track.go:495` | `handleGetTrackPoints` (GET `/tracks/{uuid}/points`) | 1000 | Frontend track detail view (`Track.tsx`) |
| `internal/pkg/api/forecast.go:125` | `handleGetTrackForecast` (GET `/tracks/{uuid}/forecast`) | 1000 | Returns per-point forecast values **indexed** into the same 1000-sample space the frontend gets from `/points` |
| `internal/pkg/forecast/summary.go:168` | `Summarizer.summarizeTrack` (background job for `track_forecasts`) | 200 | Aggregates wind sectors, avg temperature, total precipitation; not exposed via index space |
| `internal/pkg/track/point_test.go:141` | `TestSubsampleLTTB` | various | Tests only |

The two API endpoints are coupled: `/forecast` returns objects of the form
`{ index, distanceM, time, ... }` where `index` is an index into the
`/points` array. The frontend (`frontend/src/lib/time.ts`
`buildForecastTimes`) interpolates timestamps from `forecastPoints[i].index`
to every track-point index, so `/points` and `/forecast` MUST currently
agree on the indexing they expose.

### 1.2 Frontend consumers of `/points`

`pages/Track.tsx` (`pointsData?.points`) feeds the per-point `{lat, lon, ele, d}`
array to:

- `TrackMap` — uses `lat`/`lon` to draw the track and to find the nearest
  point under the cursor (`findNearest`). Hover index becomes the shared
  index in `useHoverStore`.
- `MapHoverOverlay` — uses `d` and `ele` plus `forecastTimes[idx]` for the
  bottom-left tooltip.
- `ElevationProfile` — uses `d` and `ele` to render the recharts area chart;
  cross-component hover writes back into `useHoverStore`.
- `FullscreenMapDialog` — same as `TrackMap`.
- `useForecast` — only consumes `trackPoints.length` to drive
  `buildForecastTimes(forecastPoints, trackPointsLength)`.

The `useHoverStore` index is shared between the elevation chart, the map,
and the forecast tooltip. Any replacement must produce one canonical
index space across all three.

### 1.3 Where raw `tr.Points()` (full point cloud) is used

These are not LTTB sites but are relevant because the same simplified
polylines could replace a full-blob load:

- `internal/pkg/api/track.go:1383` (`handleUploadTrack`) — runs
  `SimplifyDP` twice during upload. Already correct, no change needed.
- `internal/pkg/api/track.go:348` (`handleDownloadTrackSVG`) — passes the
  full point cloud into `Points.PreviewSVG`, which then does its own
  pixel-stride subsampling. The SVG only needs lat/lon. Today this loads
  and re-parses the blob on every miss.
- `internal/pkg/api/track.go:412` (`handleDownloadTrackProfileSVG`) — passes
  the full point cloud into `Points.ProfileSVG`. Needs lat/lon **and**
  elevation, but the encoded preview polyline carries elevation too.
- `internal/pkg/api/road_closures.go:77` — uses every point to compute
  H3 res-7 cells, then runs a fine `roadclosures.Intersects` check at
  res-12 against full lat/lon arrays. Subsampling at >~50 m would risk
  missing thin closures, so leave as-is.
- `internal/pkg/geonames/labeler.go:156` — already uses `Subsample(500 m)`.
- `internal/pkg/jobs/backfill_preview_polyline.go:133` — by definition
  needs the full points to compute the simplified versions.

## 2. Why `SubsampleLTTB()` is the wrong tool here

LTTB picks the point in each X-bucket that maximises the triangle area in
the `(distance, elevation)` plane. That is great for one-dimensional time
series (e.g. an isolated elevation chart) but it actively damages the
shape of a 2D path:

- A long straight road with monotonic elevation gain becomes very sparse
  even though small lateral kinks would be visible on the map.
- A switchback where elevation barely varies but lat/lon zig-zags
  collapses on the map because LTTB sees no "interesting" elevation
  variance.

The frontend uses these points predominantly as a 2D map polyline and
secondarily as elevation data sampled along that polyline. Douglas-Peucker
on lat/lon (already implemented as `Points.SimplifyDP`) keeps the
geometric shape correct, and the elevation values stored at each retained
point are still real samples (the encoder in
`internal/pkg/track/varint.go` already stores elevation in the precomputed
columns).

## 3. Target design

### 3.1 New helper: `Points.SimplifyForView`

Add a tiny helper in `internal/pkg/track/simplify.go`:

```go
// SimplifyForView returns a sparse view of pts suitable for client-side
// rendering: first DP-simplified by epsilonM metres of perpendicular
// distance, then thinned so consecutive points are at least minDistM
// metres apart along the track. The first and last points are always kept.
func (pts Points) SimplifyForView(epsilonM, minDistM float64) Points {
    return pts.SimplifyDP(epsilonM).Subsample(minDistM)
}
```

`Subsample` already exists. Both passes preserve the first and last point.
This is the single function backend code paths should use to derive a
"viewer" point set.

### 3.2 Index-space decoupling between `/points` and `/forecast`

The current coupling — forecast samples being keyed by an index into the
`/points` array — is what forces both endpoints to use exactly the same
subsampling. It is also fragile: if either ever uses a different N, the
frontend silently misaligns.

Replace it with **distance-based** keying:

- `/forecast` already returns `distanceM` per point. Make the frontend
  interpolate forecast timestamps by cumulative distance instead of by
  array index.
- `buildForecastTimes` becomes: for each track point at distance `d`,
  binary-search forecast points by `distanceM` and linearly interpolate
  `time` between the two flanking ones.
- This frees `/points` and `/forecast` to use independent point counts.
- Update the `/forecast` OpenAPI docs to drop the "shared index space with
  `/points`" wording. The `index` field becomes unused for the new
  client; either remove it from the response or keep it as a no-op for
  compatibility (recommend: remove, since the frontend is the only
  consumer and we control it together with the API).

### 3.3 New point counts and tolerances

Concrete recommendation for the "viewer" parameters:

| Use site | DP epsilon | min spacing | Expected count for 100 km track |
| --- | --- | --- | --- |
| `/tracks/{uuid}/points` | 5 m | 25 m | a few thousand |
| `/tracks/{uuid}/forecast` | 50 m | 500 m | ~200 |
| `forecast.Summarizer` | 50 m | 500 m | ~200 (was hardcoded 200 LTTB) |

For `/points` we want enough resolution that the elevation chart still
looks smooth and the map line looks correct when zoomed in. The 25 m
floor caps the count for very dense recordings (e.g. 1 Hz FIT files at
slow speeds) without losing visible detail.

For `/forecast` we want sample density that approximately matches the
1.1 km native ICON-CH1-EPS grid spacing; 500 m is comfortable.

Make the constants explicit, named, and documented in
`internal/pkg/track/simplify.go` next to `PreviewPolylineEpsilon{5,50}M`.

### 3.4 Use the precomputed polylines where possible

The `polyline_dp5m_varint` column already contains the result of
`SimplifyDP(5)` for every backfilled track. Decoded points carry
elevation. The following endpoints can read it instead of loading and
re-parsing the GPX/FIT blob:

1. **`/tracks/{uuid}/points`** — Decode `polyline_dp5m_varint`, run
   `Subsample(25 m)` over the result, attach cumulative distance, return.
   No blob fetch, no GPX/FIT parse. Order-of-magnitude faster on the hot
   path. Fall back to "load blob → SimplifyForView" if the column is
   `NULL` (for tracks not yet backfilled — once the backfill is removed,
   drop the fallback).
2. **`/tracks/{uuid}/preview.svg`** (`handleDownloadTrackSVG`) — Decode
   `polyline_dp50m_varint`. The SVG renderer's own pixel-stride
   downsampling already targets ~5 px segments, and 50 m DP will not be
   visible at the 16–512 px sizes this endpoint serves. Saves a blob load
   per cache miss. Fall back as above.
3. **`/tracks/{uuid}/profile.svg`** (`handleDownloadTrackProfileSVG`) —
   Decode `polyline_dp5m_varint` (we want elevation detail here, not just
   shape). Carries elevation. Same fallback rule.

`/tracks/{uuid}/forecast` cannot easily use the precomputed polyline as
the only data source because it still needs cumulative distance and
bearings, but the same "decode + SimplifyForView" path works: the
underlying polyline is already DP-simplified; we just apply the
min-distance pass on top.

`/tracks/{uuid}/road-closures` keeps loading the blob — it intentionally
runs full-resolution intersection.

Add a small helper somewhere in `internal/pkg/db` or `internal/pkg/track`:

```go
// LoadViewerPoints returns a viewer-resolution point set for a track,
// using the precomputed 5 m polyline when available and falling back to
// the raw blob otherwise.
func LoadViewerPoints(ctx context.Context, q *db.Queries, t db.Track, minDistM float64) (track.Points, error)
```

so the API handlers don't all duplicate the decode + fallback logic.

## 4. Migration steps

The order below is chosen so each step compiles, tests pass, and the
public behaviour stays correct.

1. **Add `Points.SimplifyForView`** (and tests) in
   `internal/pkg/track/simplify.go`. Pure addition.
2. **Add the viewer-points loader** described above (precomputed-first,
   blob fallback). Pure addition. Cover with a unit test that exercises
   both branches.
3. **Switch the forecast summarizer** (`internal/pkg/forecast/summary.go`)
   from `SubsampleLTTB(200, elev)` to `SimplifyForView(50, 500)`. Update
   its constants. Run the tests; the wind-sector aggregation is
   index-agnostic so this is the safest place to start.
4. **Update `/forecast` to be distance-keyed**:
   - Backend: switch `handleGetTrackForecast` to use the new viewer
     loader. Drop the `index` field from the response (or keep it as
     before but document it as deprecated). Update the OpenAPI schema.
   - Frontend: rewrite `buildForecastTimes` to interpolate by
     `distanceM` against `trackPoints[i].d` rather than by index. Update
     the `useForecast` hook signature to take the points-with-distance
     array instead of just the length. Update tests if any.
   - Leave `/points` on LTTB **for one commit** so the index spaces are
     no longer required to match before changing `/points`.
5. **Switch `/points` to use the new viewer loader**:
   - `handleGetTrackPoints` decodes `polyline_dp5m_varint`, runs
     `Subsample(25 m)`, recomputes cumulative distance, returns.
   - Bump the ETag version (`points-v3` → `points-v4`).
   - Remove `TrackPointsTarget` from `internal/pkg/api/track.go`.
6. **Switch the SVG endpoints to use the precomputed polyline** (the two
   handlers in `internal/pkg/api/track.go`). Keep the same ETag scheme
   but bump the version. The pixel-stride logic in `Points.PreviewSVG`
   and `Points.ProfileSVG` stays untouched.
7. **Remove `Points.SubsampleLTTB`** and its test
   (`internal/pkg/track/point_test.go:TestSubsampleLTTB`). At this point
   no production code references it. The golden SVG snapshot in
   `TestSubsampleLTTB/snapshot with real GPX data` is for LTTB only and
   can be deleted along with the function.
8. **Run `make check`** (full quality gate) and `npm run lint` in the
   frontend. Update the goose migration list and the OpenAPI yaml as
   required.

## 5. Risks and things to verify

- **Frontend hover sync stays valid.** The shared hover index will now
  index a smaller (a few thousand instead of 1000 LTTB) array. All four
  consumers of `trackPoints` use length-bounded indexing already
  (`hoverIndex < trackPoints.length`), so this is a no-op for them.
- **Elevation chart smoothness.** With DP+25 m, very flat sections will
  produce few interior points, but the DP keeper of "first/last" plus
  `Subsample` produces densely-spaced points wherever the path actually
  bends. Verify visually on a flat reservoir loop and on a switchback
  climb before merging.
- **Forecast time interpolation.** Distance-based interpolation needs to
  cope with `forecastPoints` having `distanceM = 0` for the first sample
  and `distanceM = totalDist` for the last. Bounds-check both ends in
  `buildForecastTimes`.
- **Backfill column nullability.** The fallback to "load blob and
  simplify" stays in place until the migration that makes
  `polyline_dp5m_varint` / `polyline_dp50m_varint` `NOT NULL` lands
  (already tracked in TODO.md). Until then, both columns can be NULL on
  freshly imported tracks before the backfill job runs.
- **OpenAPI / generated client churn.** Removing the `index` field from
  the forecast point schema regenerates `frontend/src/api/schema.gen.ts`.
  The frontend updates in step 4 must land in the same change.
- **Tests.** Add a backend test that asserts `/points` and `/forecast`
  no longer require matching counts. Add a frontend unit test for the
  new distance-based `buildForecastTimes`.

## 6. Out of scope for this work

- Making `polyline_dp5m_varint` / `polyline_dp50m_varint` `NOT NULL` and
  removing the backfill job (TODO.md item already tracked).
- Caching parsed `track.Track` objects (TODO.md "potentially cache track
  blob -> track.Track{} for performance"). The viewer-points loader
  already shortcuts the hot paths without a parse-cache.
- Touching `road_closures.go`'s point usage; intentional full-resolution.
