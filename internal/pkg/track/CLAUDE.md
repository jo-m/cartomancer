## Track points pipeline

Track points exist in three forms:

1. **Original blob** - uploaded GPX/FIT bytes, zstd-compressed in `blobs`. Cross-user dedup by SHA-256 (`blob.Create`, `blob.Get`). Re-parsed only by the upload handler, the road-closures endpoint, and the data export.
2. **Simplified varint polylines** on the `tracks` row, two `NOT NULL` BLOB columns:
   - `polyline_dp5m_varint`  - DP-simplified at 5 m. Used by `/tracks/{uuid}/points`, `/tracks/{uuid}/profile.svg`, `/tracks/polylines/5m`.
   - `polyline_dp50m_varint` - DP-simplified at 50 m. Used by `/tracks/{uuid}/preview.svg`, `/tracks/polylines/50m`, and as input to `/tracks/{uuid}/forecast` and the forecast summarizer (interpolated at fixed 200 m / 500 m steps).
   Encoding: per-point zig-zag varint deltas of (lat, lon, elevation, distance). lat/lon/elevation precision 1e5 (~1.1 m / 1 cm); distance precision 1 m. See `internal/pkg/track/varint.go`.
3. **Frontend-shaped JSON** - handlers decode the relevant column and apply `Subsample` (per-endpoint min-distance threshold from `simplify.go`) before responding. Hot paths never re-parse the original blob.

Polylines are computed inline at upload time (`handleUploadTrack`).

`/tracks/{uuid}/points` and `/tracks/{uuid}/forecast` are **distance-keyed** (each point carries `distanceM`); the frontend interpolates by cumulative distance, not array index, so the two endpoints can use independent point counts.

Changing the varint format requires repopulating both columns for all existing rows in a migration (see `00000000000019_track_preview_polyline_distance.sql` for the null-out pattern + backfill job).
