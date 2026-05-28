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

All sources share the same structure: `Fetch(ctx)` client + `Downloader` job with `MinRefreshAge = 23h` early-return guard on `GetLatestRoadClosureCreatedAt(jobKind)`, registered as periodic in `main.go`. WFS sources use the shared `internal/pkg/wfs/gml` decoder.

- `astra/` - geo.admin.ch MapServer `find`, layer `ch.astra.veloland-sperrungen_umleitungen`. JSON. `Type` from `sperrungen_type` (`detour`/`closed_way`).
- `zh/` - Canton Zurich WFS (`maps.zh.ch/wfs/TbaBaustellenZHWFS`, layer `ms:baustellen-detailansicht`). Filtered to `aktiv*`/`zukünftig*` (`status_baustelle`). `Type` hardcoded `closed_way`.
- `sz/` - Canton Schwyz WFS (`map.geo.sz.ch/mapserv_proxy`, layer `ms:ch.sz.a083a.baustellen`). No status filter. German display-string dates parsed best-effort by `date.go`, NULL on failure. No `gml:id`, so `sourceID()` derives a SHA-1 fingerprint from feature text + first geometry vertex. `Type` hardcoded `closed_way`.
- `sg/` - Canton St. Gallen WFS (`stgallen.opendatasoft.com`, dataset `baustellenkoordination`). No status filter. `Type` derived from lowercased title (`bew`) + description (`adresse`): contains "sperrung"/"gesperrt" -> `closed_way`, else `detour`.
- `ag/` - Canton Aargau ArcGIS REST MapServer (`arcgis.geo.ag.ch/.../ATB/Baustellen_online/MapServer/0`). GeoJSON output (`f=geojson`, `outSR=4326`); server-side filter `tDate >= CURRENT_TIMESTAMP` keeps the payload to currently or future-active sites. `SourceID` is `ag-<OBJECTID>`. Dates decode either as ISO-8601 strings or epoch-ms numbers via the `apiDate` custom unmarshaller. `Type` derived from `Bezeichnung` + `Behinderung_Karte`/`Behinderung_Tabelle` with the same "sperrung"/"gesperrt" heuristic as `sg/`.

### Adding a new source

1. New subpackage `internal/pkg/roadclosures/<src>/`.
2. Client `Fetch(ctx)` returning normalized features. Provide `DataAttribution attribute.Attribution`.
3. `Downloader` implementing `jobs.Job[DownloaderArgs]`; in `Run`, gate on `MinRefreshAge`, then do delete-then-insert in one `WithTx`.
4. Per-feature: build `roadclosures.ClosureInsert{Type: "detour"|"closed_way", ...}` and call `roadclosures.Insert`. Use `roadclosures.NullString` for optional text.
5. Register in `main.go` (`MustRegisterJob` + `jobs.Periodic`).
6. Add `<src>.DataAttribution` to the `Attributions` slice in `internal/pkg/api/version.go` so the source appears on the /about page.

### Tests

- `*_online_test.go` hits live upstream endpoints; gated behind the `online` build tag.
- Unit tests cover GML parsing (`zh/gml_test.go`), feature decoding, cell extraction, and intersection.
