## Track points pipeline

Track points flow through the backend in three forms:

1. **Original blob** — the uploaded GPX/FIT bytes, zstd-compressed in the
   `blobs` table. Deduplicated cross-user by SHA-256 (`blob.Create`,
   `blob.Get`). Only re-parsed by the upload handler, the road-closures
   endpoint, and the data export.
2. **Simplified varint polylines** stored on the `tracks` row as two
   `NOT NULL` BLOB columns:
   - `polyline_dp5m_varint` — Douglas-Peucker simplified at 5 m, used by
     `/tracks/{uuid}/points`, `/tracks/{uuid}/profile.svg`, and the
     `/tracks/polylines/5m` bulk listing.
   - `polyline_dp50m_varint` — DP-simplified at 50 m, used by
     `/tracks/{uuid}/preview.svg`, `/tracks/polylines/50m`, and as the
     input to `/tracks/{uuid}/forecast` and the forecast summarizer
     (interpolated at fixed 200 m / 500 m steps).
   Encoding is per-point zig-zag varint deltas of (lat, lon, elevation,
   distance). lat/lon/elevation precision 1e5 (~1.1 m / 1 cm); distance
   precision 1 m. See `internal/pkg/track/varint.go`.
3. **Frontend-shaped JSON** — handlers decode the relevant column and
   apply `Subsample` (with the per-endpoint min-distance threshold from
   `internal/pkg/track/simplify.go`) before responding. Hot paths never
   re-parse the original blob.

The polylines are computed inline at upload time (`handleUploadTrack`).

`/tracks/{uuid}/points` and `/tracks/{uuid}/forecast` use **distance-keyed**
indexing (each response carries `distanceM` per point); the frontend
interpolates by cumulative distance, not by array index, so the two
endpoints can use independent point counts.

When changing the varint format, both columns must be repopulated for
all existing rows in a migration (see migration
`00000000000019_track_preview_polyline_distance.sql` for the pattern of
nulling them out, paired with a one-shot backfill job that has since
been removed).
