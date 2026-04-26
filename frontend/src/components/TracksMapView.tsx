import { useEffect, useMemo, useRef, useState } from "react"
import { Link, useNavigate } from "react-router-dom"
import OlMap from "ol/Map"
import OlView from "ol/View"
import TileLayer from "ol/layer/Tile"
import VectorTileLayer from "ol/layer/VectorTile"
import VectorLayer from "ol/layer/Vector"
import VectorSource from "ol/source/Vector"
import WMTS from "ol/source/WMTS"
import Feature from "ol/Feature"
import { LineString } from "ol/geom"
import { Stroke, Style } from "ol/style"
import { isEmpty as extentIsEmpty } from "ol/extent"
import { toLonLat } from "ol/proj"
import { getLV95TileGrid, getLV95ViewConfig } from "@swissgeo/coordinates/ol"
import { PMTilesVectorSource } from "ol-pmtiles"
import { $api } from "../api/client"
import { lv95, projectPoint } from "../lib/proj"
import { selectMapLayer, unionBbox } from "../lib/mapLayer"
import type { Bbox, MapLayer } from "../lib/mapLayer"
import { createPmtilesStyleFn } from "../lib/pmtilesStyle"
import { trackColorFromUUID } from "../lib/trackColor"
import SvgIcon from "../assets/SvgIcon"
import distanceSvg from "../assets/distance.svg?raw"
import elevationSvg from "../assets/elevation.svg?raw"
import temperatureSvg from "../assets/temperature.svg?raw"
import rainSvg from "../assets/rain.svg?raw"
import MiniWindRose from "./MiniWindRose"
import MapAttribution from "./MapAttribution"
import { formatDistance, formatAscent } from "../lib/format"
import { stringParam, useUrlState } from "../hooks/useUrlState"

import "ol/ol.css"

interface PopoverForecast {
  avgTemperatureC?: number | null
  totalPrecipitationMm?: number | null
  windHeadMs?: number
  windRightMs?: number
  windTailMs?: number
  windLeftMs?: number
}

/**
 * Query parameters for the /tracks/polylines/50m endpoint. Pagination is
 * intentionally absent: the endpoint applies a server-side cap.
 */
export type TracksPolylinesQuery = NonNullable<
  Parameters<typeof $api.useQuery<"get", "/tracks/polylines/50m">>[2]
>["params"] extends infer P
  ? P extends { query?: infer Q }
    ? Q
    : never
  : never

interface TracksMapViewProps {
  query: TracksPolylinesQuery
  selectionActive: boolean
  selected: Set<string>
  onSelect: (e: React.MouseEvent, uuid: string, index: number) => void
}

interface PopoverState {
  uuid: string
  name: string
  userName: string
  totalDistanceM: number
  totalAscentM: number
  forecast: PopoverForecast | null
  pixel: [number, number]
}

/**
 * Default style for an unselected, unhovered track. The white halo provides
 * contrast against any basemap; the inner stroke uses the track's stable
 * UUID-derived color so adjacent tracks remain distinguishable.
 */
function makeBaseStyle(color: string): Style[] {
  return [
    new Style({
      stroke: new Stroke({ color: "rgba(255,255,255,0.85)", width: 4 }),
    }),
    new Style({ stroke: new Stroke({ color, width: 2 }) }),
  ]
}

const hoverStyle = [
  new Style({ stroke: new Stroke({ color: "#ffffff", width: 6 }) }),
  new Style({ stroke: new Stroke({ color: "#f59e0b", width: 3 }) }),
]

const selectedStyle = [
  new Style({ stroke: new Stroke({ color: "#ffffff", width: 5 }) }),
  new Style({ stroke: new Stroke({ color: "#22c55e", width: 3 }) }),
]

/** Returns the base style for a track, looking up its UUID-derived color. */
function baseStyleFor(uuid: string): Style[] {
  return makeBaseStyle(trackColorFromUUID(uuid))
}

/**
 * Parses the m URL parameter shape "lat,lon,zoom" used to persist map
 * viewport between sessions. Returns null on invalid input.
 */
function parseViewport(
  raw: string
): { lat: number; lon: number; zoom: number } | null {
  const parts = raw.split(",").map(Number)
  if (parts.length !== 3 || !parts.every(Number.isFinite)) return null
  const [lat, lon, zoom] = parts
  if (lat < -90 || lat > 90 || lon < -180 || lon > 180) return null
  return { lat, lon, zoom }
}

const viewportSchema = {
  m: stringParam(),
}

/**
 * Renders the map view of tracks: every track returned by /tracks/polylines
 * becomes a polyline on the map. Hovering a track shows a popover with
 * stats and a preview SVG; clicking it navigates to the track detail page,
 * or toggles selection when the parent has selection mode active.
 *
 * The map viewport (center + zoom) is persisted in the URL via the m=
 * parameter so that switching back from the list view restores the
 * previous map position.
 */
export default function TracksMapView({
  query,
  selectionActive,
  selected,
  onSelect,
}: TracksMapViewProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<HTMLDivElement>(null)
  const mapInstanceRef = useRef<OlMap | null>(null)
  const sourceRef = useRef<VectorSource | null>(null)
  const featureMapRef = useRef<Map<string, Feature>>(new Map())
  const hoveredFeatureRef = useRef<Feature | null>(null)
  const navigate = useNavigate()
  const [popover, setPopover] = useState<PopoverState | null>(null)
  const [viewportUrl, setViewportUrl] = useUrlState(viewportSchema)
  const [tileErrorUrl, setTileErrorUrl] = useState<string | null>(null)
  const [darkMode, setDarkMode] = useState(
    () => window.matchMedia("(prefers-color-scheme: dark)").matches
  )

  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = (e: MediaQueryListEvent) => setDarkMode(e.matches)
    mq.addEventListener("change", handler)
    return () => mq.removeEventListener("change", handler)
  }, [])

  const { data, isLoading, error } = $api.useQuery(
    "get",
    "/tracks/polylines/50m",
    {
      params: { query },
    }
  )

  const { data: mapsData } = $api.useQuery("get", "/maps")

  // Compute union bbox of all returned tracks so we can pick the best layer.
  const unionedBbox = useMemo<Bbox | null>(() => {
    if (!data) return null
    let acc: Bbox | null = null
    for (const t of data.tracks) {
      if (!t.bounds) continue
      acc = unionBbox(acc, {
        minLat: t.bounds.min.lat,
        maxLat: t.bounds.max.lat,
        minLon: t.bounds.min.lon,
        maxLon: t.bounds.max.lon,
      })
    }
    return acc
  }, [data])

  const layer: MapLayer = useMemo(() => {
    if (!unionedBbox) return { type: "none" }
    return selectMapLayer(unionedBbox, mapsData ?? [])
  }, [unionedBbox, mapsData])

  const tileError = layer.type === "pmtiles" && tileErrorUrl === layer.url

  // (Re)build the map whenever the layer choice changes. We keep the
  // VectorSource across data updates so we don't recreate the whole map
  // when only the track set changes.
  useEffect(() => {
    if (!mapRef.current) return

    const source = new VectorSource()
    sourceRef.current = source
    featureMapRef.current = new Map()

    const layers = []
    if (layer.type === "swisstopo") {
      const tileGrid = getLV95TileGrid()
      layers.push(
        new TileLayer({
          source: new WMTS({
            url: "https://wmts.geo.admin.ch/1.0.0/ch.swisstopo.pixelkarte-farbe/default/current/2056/{TileMatrix}/{TileCol}/{TileRow}.jpeg",
            tileGrid,
            projection: lv95,
            requestEncoding: "REST",
            layer: "ch.swisstopo.pixelkarte-farbe",
            style: "default",
            matrixSet: "2056",
          }),
        })
      )
    } else if (layer.type === "pmtiles") {
      const pmtilesSource = new PMTilesVectorSource({ url: layer.url })
      pmtilesSource.on("error", () => setTileErrorUrl(layer.url))
      layers.push(
        new VectorTileLayer({
          source: pmtilesSource,
          style: createPmtilesStyleFn(darkMode),
        })
      )
    }

    layers.push(new VectorLayer({ source }))

    const view =
      layer.type === "swisstopo"
        ? new OlView(getLV95ViewConfig())
        : new OlView({ center: [0, 0], zoom: 2 })

    const map = new OlMap({
      target: mapRef.current,
      layers,
      view,
    })

    // Restore persisted viewport, if any (only when not Swiss layer).
    if (layer.type !== "swisstopo" && viewportUrl.m) {
      const vp = parseViewport(viewportUrl.m)
      if (vp) {
        view.setCenter(projectPoint(vp.lon, vp.lat, layer.type))
        view.setZoom(vp.zoom)
      }
    }

    mapInstanceRef.current = map
    return () => {
      map.setTarget(undefined)
      mapInstanceRef.current = null
      sourceRef.current = null
      featureMapRef.current = new Map()
    }
    // viewportUrl.m is intentionally only read on (re)mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [layer, darkMode])

  // Sync features into the source when data or layer changes.
  useEffect(() => {
    const source = sourceRef.current
    const map = mapInstanceRef.current
    if (!source || !map || !data) return

    source.clear()
    const featureMap = new Map<string, Feature>()
    for (let i = 0; i < data.tracks.length; i++) {
      const t = data.tracks[i]
      if (t.polyline.length < 2) continue
      const coords = t.polyline.map(([lat, lon]) =>
        projectPoint(lon, lat, layer.type)
      )
      const f = new Feature({ geometry: new LineString(coords) })
      f.setId(t.uuid)
      f.set("trackUuid", t.uuid)
      f.set("trackName", t.name)
      f.set("trackUserName", t.userName)
      f.set("trackTotalDistanceM", t.totalDistanceM)
      f.set("trackTotalAscentM", t.totalAscentM)
      f.set("trackForecast", t.forecast ?? null)
      f.set("trackIndex", i)
      f.setStyle(selected.has(t.uuid) ? selectedStyle : baseStyleFor(t.uuid))
      source.addFeature(f)
      featureMap.set(t.uuid, f)
    }
    featureMapRef.current = featureMap

    // Fit on first load when no persisted viewport.
    if (!viewportUrl.m) {
      const extent = source.getExtent()
      if (extent && !extentIsEmpty(extent)) {
        map.getView().fit(extent, { padding: [40, 40, 40, 40], maxZoom: 14 })
      }
    }
    // viewportUrl.m only matters on initial fit.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data, layer, selected])

  // Persist viewport into URL on move-end.
  useEffect(() => {
    const map = mapInstanceRef.current
    if (!map || layer.type === "swisstopo") return
    const view = map.getView()
    const onMoveEnd = () => {
      const c = view.getCenter()
      const z = view.getZoom()
      if (!c || z == null) return
      const [lon, lat] = toLonLat(c)
      setViewportUrl({
        m: `${lat.toFixed(5)},${lon.toFixed(5)},${z.toFixed(2)}`,
      })
    }
    map.on("moveend", onMoveEnd)
    return () => {
      map.un("moveend", onMoveEnd)
    }
  }, [layer, setViewportUrl])

  // Hover and click: track features only.
  useEffect(() => {
    const map = mapInstanceRef.current
    const container = containerRef.current
    if (!map || !container) return

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onMove = (evt: any) => {
      const pixel = evt.pixel as [number, number]
      let hovered: Feature | null = null
      map.forEachFeatureAtPixel(
        pixel,
        (f) => {
          const feat = f as Feature
          if (!feat.get("trackUuid")) return false
          hovered = feat
          return true
        },
        { hitTolerance: 4 }
      )

      // Sticky popover: only react to transitions to a new feature. When the
      // cursor leaves a polyline (without reaching another) the popover and
      // the feature highlight stay put, so the user can move the cursor over
      // the popover content. The popover is dismissed only when leaving the
      // whole map container or moving onto a different track.
      if (hovered && hovered !== hoveredFeatureRef.current) {
        if (hoveredFeatureRef.current) {
          const prev = hoveredFeatureRef.current
          const uuid = prev.get("trackUuid") as string
          prev.setStyle(selected.has(uuid) ? selectedStyle : baseStyleFor(uuid))
        }
        const h = hovered as Feature
        h.setStyle(hoverStyle)
        hoveredFeatureRef.current = h
        setPopover({
          uuid: h.get("trackUuid") as string,
          name: h.get("trackName") as string,
          userName: h.get("trackUserName") as string,
          totalDistanceM: h.get("trackTotalDistanceM") as number,
          totalAscentM: h.get("trackTotalAscentM") as number,
          forecast: h.get("trackForecast") as PopoverForecast | null,
          pixel,
        })
      }
      map.getTargetElement().style.cursor = hovered ? "pointer" : ""
    }

    const onLeave = () => {
      if (hoveredFeatureRef.current) {
        const prev = hoveredFeatureRef.current
        const uuid = prev.get("trackUuid") as string
        prev.setStyle(selected.has(uuid) ? selectedStyle : baseStyleFor(uuid))
        hoveredFeatureRef.current = null
      }
      setPopover(null)
      map.getTargetElement().style.cursor = ""
    }

    map.on("pointermove", onMove)
    // Listen on the outer container rather than the OL viewport so that
    // moving the cursor onto the popover (a sibling of the viewport but
    // inside the container) does not dismiss it.
    container.addEventListener("pointerleave", onLeave)
    return () => {
      map.un("pointermove", onMove)
      container.removeEventListener("pointerleave", onLeave)
    }
    // layer and darkMode are included because the OL map is rebuilt when
    // either changes, and the listeners must be re-attached to the new map.
  }, [selected, layer, darkMode])

  // Click handler: in selection mode, toggle selection; otherwise navigate
  // to the clicked track. We attach it on the map so any click on a feature
  // counts even when the popover isn't pinned.
  useEffect(() => {
    const map = mapInstanceRef.current
    if (!map) return
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onClick = (evt: any) => {
      const pixel = evt.pixel as [number, number]
      let hit = false
      map.forEachFeatureAtPixel(
        pixel,
        (f) => {
          const feat = f as Feature
          const uuid = feat.get("trackUuid") as string | undefined
          const idx = feat.get("trackIndex") as number | undefined
          if (!uuid || idx == null) return false
          hit = true
          if (selectionActive) {
            const me = evt.originalEvent as React.MouseEvent
            onSelect(me, uuid, idx)
          } else {
            navigate(`/tracks/${uuid}`)
          }
          return true
        },
        { hitTolerance: 4 }
      )
      if (!hit) {
        // Clicking on empty map dismisses the sticky popover and clears the
        // associated feature highlight.
        if (hoveredFeatureRef.current) {
          const prev = hoveredFeatureRef.current
          const uuid = prev.get("trackUuid") as string
          prev.setStyle(selected.has(uuid) ? selectedStyle : baseStyleFor(uuid))
          hoveredFeatureRef.current = null
        }
        setPopover(null)
      }
    }
    map.on("click", onClick)
    return () => {
      map.un("click", onClick)
    }
    // layer and darkMode trigger a map rebuild, requiring re-attachment.
  }, [selectionActive, onSelect, navigate, layer, darkMode, selected])

  return (
    <div
      ref={containerRef}
      className="relative h-[calc(100vh-260px)] min-h-[400px] w-full rounded-lg border border-border"
    >
      {layer.type === "none" && (
        <div
          className="absolute inset-0 rounded-lg"
          style={{ background: darkMode ? "#24201a" : "#f0ece4" }}
        />
      )}
      <div ref={mapRef} className="h-full w-full" />

      {isLoading && (
        <div className="pointer-events-none absolute left-2 top-2 rounded bg-panel/80 px-2 py-1 text-xs text-text-muted ring-1 ring-border">
          Loading tracks...
        </div>
      )}
      {error && (
        <div
          role="alert"
          className="absolute left-2 top-2 rounded bg-panel/90 px-3 py-1.5 text-sm text-error ring-1 ring-border"
        >
          {error.message}
        </div>
      )}
      {tileError && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <span className="rounded bg-panel/90 px-3 py-1.5 text-sm text-error ring-1 ring-border">
            Map tiles unavailable
          </span>
        </div>
      )}

      <MapAttribution layer={layer} />

      {data && (
        <div className="absolute bottom-2 left-2 max-w-[80%] rounded bg-panel/95 px-3 py-1.5 text-xs text-text ring-1 ring-border">
          {data.totalCount > data.tracks.length ? (
            <>
              Showing {data.tracks.length} of {data.totalCount} matching tracks
              (cap {data.limit}). Refine filters to see fewer.
            </>
          ) : (
            <>
              Showing {data.tracks.length} matching track
              {data.tracks.length === 1 ? "" : "s"} (cap {data.limit}).
            </>
          )}
          {data.pendingCount > 0 && (
            <> {data.pendingCount} still being processed.</>
          )}
        </div>
      )}

      {popover && (
        <div
          className="pointer-events-auto absolute z-10 w-56 rounded bg-panel/95 p-2 text-xs shadow-md ring-1 ring-border"
          style={{
            left: popover.pixel[0] + 12,
            top: popover.pixel[1] + 12,
          }}
        >
          <p className="truncate font-medium text-text">{popover.name}</p>
          <p className="text-text-muted">{popover.userName}</p>
          <div className="mt-0.5 flex items-center gap-2 text-text-muted">
            <span className="flex items-center gap-0.5">
              <SvgIcon svg={distanceSvg} className="inline h-3 w-3" />
              {formatDistance(popover.totalDistanceM)}
            </span>
            <span className="flex items-center gap-0.5">
              <SvgIcon svg={elevationSvg} className="inline h-3 w-3" />
              {formatAscent(popover.totalAscentM)}
            </span>
          </div>
          {popover.forecast && (
            <div className="mt-1 flex items-center gap-x-2">
              {popover.forecast.avgTemperatureC != null && (
                <span className="flex items-center gap-0.5 text-error">
                  <SvgIcon svg={temperatureSvg} className="inline h-3 w-3" />
                  {popover.forecast.avgTemperatureC.toFixed(0)}
                  &deg;C
                </span>
              )}
              {popover.forecast.totalPrecipitationMm != null && (
                <span className="flex items-center gap-0.5 text-info">
                  <SvgIcon svg={rainSvg} className="inline h-3 w-3" />
                  {popover.forecast.totalPrecipitationMm < 0.1
                    ? "dry"
                    : `${popover.forecast.totalPrecipitationMm.toFixed(1)} mm`}
                </span>
              )}
              <MiniWindRose
                head={popover.forecast.windHeadMs}
                right={popover.forecast.windRightMs}
                tail={popover.forecast.windTailMs}
                left={popover.forecast.windLeftMs}
              />
            </div>
          )}
          <Link
            to={`/tracks/${popover.uuid}`}
            className="mt-1 block text-primary hover:underline"
          >
            View track
          </Link>
        </div>
      )}
    </div>
  )
}
