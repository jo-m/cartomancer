## Streetlights

Ingests public street lighting (lamp points) from multiple upstream sources.

### Pipeline

1. Per-source downloader job (subpackages, see below) fetches features and converts each to a `streetlights.StreetlightInsert`.
2. `streetlights.Insert(ctx, tx, s, now)` writes one `streetlights` row plus its H3 res-9 cell index. Same-cycle rows share `now`. Nil geometry -> `ErrNilGeometry`; non-point geometries -> `ErrNonPointGeometry`; callers should skip such features upstream.
3. Each cycle is one tx: `DeleteStreetlightsByInsertedBy(jobKind)` then insert. `inserted_by == jobKind` scopes deletes per source.
4. Per-lamp source attributes are stored verbatim in the `properties` JSON column; empty, missing and JSON null properties become `{}`.

### H3 resolution (`constants.go`)

- `CellResolution = 9` - matches DB index `streetlight_cells_res9`. Finer than roadclosures' resolution 7 because lamps are dense point features: one res-7 cell can contain well over a thousand lamps in the city of Zurich.

### Sources (subpackages)

All sources share the same structure: `Fetch(ctx)` client + `Downloader` job with a `MinRefreshAge = 30d` early-return guard on `GetLatestStreetlightCreatedAt(jobKind)`, registered as a daily periodic job in `main.go` (the gate throttles refresh to monthly; the daily tick lets a transient failure self-heal). Both sources fetch through `wfs.GetFeatureGeoJSON`.

- `ktzh/` - Canton Zurich OGD WFS (`maps.zh.ch/wfs/OGDZHWFS`, layer `ms:ogd-0124_giszhpub_tba_str_beleuchtung_p`), WFS 2.0.0 via `wfs.NewClient`. Features have no id, so `SourceID` is `ktzh-<geodb_oid>`. `srsName` must use the EPSG URN form.
- `stadtzh/` - City of Zurich WFS (`www.ogd.stadt-zuerich.ch/wfs/geoportal/Oeffentliche_Beleuchtung_der_Stadt_Zuerich`, layer `ewz_brennstelle_p`), WFS 1.1.0 only, via `wfs.NewClientV11` (2.0.0 requests fail). `SourceID` is `stadtzh-<feature id>`, falling back to `objectid`. `srsName` must use the short `EPSG:4326` form.

### Adding a new source

1. New subpackage `internal/pkg/streetlights/<src>/`.
2. Client `Fetch(ctx)` returning normalized features. Provide `DataAttribution attribute.Attribution`.
3. `Downloader` implementing `jobs.Job[DownloaderArgs]`; in `Run`, gate on `MinRefreshAge`, then do delete-then-insert in one `WithTx`.
4. Per-feature: build `streetlights.StreetlightInsert{...}` and call `streetlights.Insert`. Use `streetlights.NullString` for optional text.
5. Register in `main.go` (`MustRegisterJob` + `jobs.Periodic`).
6. Add `<src>.DataAttribution` to the `Attributions` slice in `internal/pkg/api/version.go` so the source appears on the /about page.

### Tests

- `*_online_test.go` hits live upstream endpoints; gated behind the `online` build tag.
- Unit tests cover feature decoding and source id derivation.
