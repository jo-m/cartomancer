## Streetlights

Ingests public street lighting (lamp points) from upstream WFS sources.

### Pipeline

- Each source subpackage has a downloader job that fetches features and calls `streetlights.Insert(ctx, tx, s, now)` per lamp, sharing one timestamp per cycle. `Insert` rejects nil/non-point geometries before any writes ([ErrNilGeometry], [ErrNonPointGeometry]); jobs skip those features.
- Each cycle is one tx: `DeleteStreetlightsByInsertedBy(jobKind)` then insert; `inserted_by` scopes deletes per source.
- A row keeps position (indexed H3 res-9 `cell` plus GeoJSON geometry), source id, and an FK to `streetlight_attributions`, the normalized (attribution, attribution_href) lookup. No properties column.

### Track coverage (`coverage.go`)

`ComputeTrackCoverage(ctx, d, t)` samples the dp5m polyline every `SampleStepM = 5 m`; a sample is lit when a lamp is within `LitRadiusM = 20 m`. Candidate lamps come from `h3.GridDisk(cell, 1)` of the res-9 sample cell (superset: edge ~200 m; `{cell}` fallback for pentagons), pre-filtered by an equirectangular box before the exact distance. Boundaries land on midpoints between the last lit and first unlit sample (+/-2.5 m). Attributions list only sources of lamps that lit something, deduped and sorted.

### Sources (subpackages)

- `ktzh/` (Canton Zurich, `maps.zh.ch/wfs/OGDZHWFS`, layer `ms:ogd-0124_giszhpub_tba_str_beleuchtung_p`): WFS 2.0.0 via `wfs.NewClient`; `srsName` as EPSG URN; no feature ids, so `SourceID` is `ktzh-<geodb_oid>`.
- `stadtzh/` (City of Zurich, `www.ogd.stadt-zuerich.ch/wfs/geoportal/Oeffentliche_Beleuchtung_der_Stadt_Zuerich`, layer `ewz_brennstelle_p`): WFS 1.1.0 only via `wfs.NewClientV11`; `srsName` `EPSG:4326`; `SourceID` is `stadtzh-<feature id>` with `objectid` fallback.

Both are daily periodic jobs with a `MinRefreshAge = 30d` gate on `GetLatestStreetlightCreatedAt(jobKind)`: the daily tick lets transient failures self-heal, the gate throttles refreshes to monthly.

### Adding a source

Subpackage with `Fetch(ctx)` and a `DataAttribution`, plus a downloader job that does delete-then-insert in one `WithTx` via `Insert`; register in `main.go` (`MustRegisterJob` + `jobs.Periodic`); add `DataAttribution` to `Attributions` in `internal/pkg/api/version.go`.

### Tests

Unit tests cover decoding, source ids and the insert path. `*_online_test.go` hits live endpoints behind the `online` build tag.
