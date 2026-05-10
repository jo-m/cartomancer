## Road closures

Ingests road closures and detours from multiple upstream sources, intersects them with track points.

### Pipeline

1. Per-source downloader job (subpackages, see below) fetches features and converts each to a `roadclosures.ClosureInsert`.
2. `roadclosures.Insert(ctx, tx, c, now)` writes one `road_closures` row plus its H3 res-7 cell index. Same-cycle rows share `now`. Nil geometry -> `ErrNilGeometry`; callers should skip such features upstream.
3. Each cycle is one tx: `DeleteRoadClosuresByInsertedBy(jobKind)` then insert. `inserted_by == jobKind` scopes deletes per source.
4. API (`internal/pkg/api/road_closures.go`) looks up candidates with `GetActiveRoadClosuresByCells` (track points -> res-7 cells), then confirms with `roadclosures.Intersects(geometryJSON, lats, lons)` at res 12 (no interpolation on the track side).

### H3 resolutions (`constants.go`)

- `CellResolution = 7`  - coarse, matches DB index `road_closure_cells_res7`.
- `FineResolution = 12` - confirmation check in `Intersects`.

Closure geometries are interpolated between vertices at half the hex edge length (see `addPoints` in `cells.go`); track points are not.

### Sources (subpackages)

- `astra/` - geo.admin.ch MapServer `find`, layer `ch.astra.veloland-sperrungen_umleitungen`. JSON. Job kind `roadclosures.astra.downloader`. `Type` taken from `sperrungen_type` (`detour` / `closed_way`).
- `zh/`    - Canton Zurich WFS (`maps.zh.ch/wfs/TbaBaustellenZHWFS`, layer `ms:baustellen-detailansicht`). GML 3.2 via custom decoder in `gml.go`. Job kind `roadclosures.zh.downloader`. Filtered to `aktiv*` / `zukünftig*` statuses. `Type` hardcoded to `closed_way`.

Both downloaders share the same structure: `Fetch(ctx)` client + `Downloader` job with `MinRefreshAge = 23h` early-return guard on `GetLatestRoadClosureCreatedAt(jobKind)`, registered as periodic in `main.go`.

### Adding a new source

1. New subpackage `internal/pkg/roadclosures/<src>/`.
2. Client `Fetch(ctx)` returning normalized features. Provide `DataAttribution attribute.Attribution`.
3. `Downloader` implementing `jobs.Job[DownloaderArgs]`; in `Run`, gate on `MinRefreshAge`, then do delete-then-insert in one `WithTx`.
4. Per-feature: build `roadclosures.ClosureInsert{Type: "detour"|"closed_way", ...}` and call `roadclosures.Insert`. Use `roadclosures.NullString` for optional text.
5. Register in `main.go` (`MustRegisterJob` + `jobs.Periodic`).

### Tests

- `*_online_test.go` hits live upstream endpoints; gated behind the `online` build tag.
- Unit tests cover GML parsing (`zh/gml_test.go`), feature decoding, cell extraction, and intersection.
