import { useEffect, useRef, useCallback, useMemo, useState, memo } from "react"
import OlMap from "ol/Map"
import OlView from "ol/View"
import Overlay from "ol/Overlay"
import TileLayer from "ol/layer/Tile"
import VectorTileLayer from "ol/layer/VectorTile"
import VectorLayer from "ol/layer/Vector"
import VectorSource from "ol/source/Vector"
import WMTS from "ol/source/WMTS"
import Feature from "ol/Feature"
import GeoJSON from "ol/format/GeoJSON"
import { isEmpty as extentIsEmpty } from "ol/extent"
import { LineString, Point } from "ol/geom"
import { Circle, Fill, Stroke, Style, Icon } from "ol/style"
import { getLV95TileGrid, getLV95ViewConfig } from "@swissgeo/coordinates/ol"
import { lv95, projectPoint } from "../lib/proj"
import { PMTilesVectorSource } from "ol-pmtiles"

import type { MapLayer } from "../lib/mapLayer"
import { createPmtilesStyleFn } from "../lib/pmtilesStyle"
import {
  TRACK_LINE_HALO_WIDTH,
  TRACK_LINE_INNER_WIDTH,
} from "../lib/trackMapStyle"
import {
  ARROW_SCREEN_SPACING_PX,
  buildArrowHaloCanvas,
  buildArrowInnerCanvas,
  buildDetourSignCanvas,
  buildEndStyleCanvas,
} from "../lib/mapArrows"
import type { HoverStore } from "../hooks/useHoverSync"
import type { RoadClosure } from "../types/map"
import { fmtDate } from "../lib/time"
import MapAttribution from "./MapAttribution"

import "ol/ol.css"

const markerHiddenStyle = new Style({})

const startStyle = new Style({
  image: new Circle({
    radius: 7,
    fill: new Fill({ color: "#22c55e" }),
    stroke: new Stroke({ color: "#ffffff", width: 2 }),
  }),
})

const endStyle = new Style({
  image: new Icon({
    img: buildEndStyleCanvas(),
    anchor: [0.5, 0.5],
  }),
})

interface TrackPoint {
  lat: number
  lon: number
  ele: number
  d: number
}

const detourStyle = new Style({
  stroke: new Stroke({ color: "rgba(245, 158, 11, 0.7)", width: 10 }),
  image: new Icon({
    img: buildDetourSignCanvas("rgba(245, 158, 11, 0.85)", "rgb(0 0 0)"),
    anchor: [0.5, 0.5],
  }),
})

const detourStyleHover = new Style({
  stroke: new Stroke({ color: "rgba(245, 158, 11, 0.9)", width: 10 }),
  image: new Icon({
    img: buildDetourSignCanvas("rgba(245, 158, 11, 1)", "rgb(0 0 0)"),
    anchor: [0.5, 0.5],
  }),
})

const closureStyle = [
  new Style({
    stroke: new Stroke({
      color: "rgba(220, 38, 38, 0.7)",
      width: 10,
      lineDash: [14, 14],
      lineCap: "butt",
    }),
    image: new Circle({
      radius: 10,
      fill: new Fill({ color: "rgba(255, 255, 255, 0.85)" }),
      stroke: new Stroke({ color: "rgba(220, 38, 38, 0.85)", width: 4 }),
    }),
  }),
  new Style({
    stroke: new Stroke({
      color: "rgba(255, 255, 255, 0.7)",
      width: 10,
      lineDash: [14, 14],
      lineDashOffset: 14,
      lineCap: "butt",
    }),
  }),
]

const closureStyleHover = [
  new Style({
    stroke: new Stroke({
      color: "rgba(220, 38, 38, .9)",
      width: 10,
      lineDash: [14, 14],
      lineCap: "butt",
    }),
    image: new Circle({
      radius: 10,
      fill: new Fill({ color: "rgba(255, 255, 255, 1)" }),
      stroke: new Stroke({ color: "rgba(220, 38, 38, 1)", width: 4 }),
    }),
  }),
  new Style({
    stroke: new Stroke({
      color: "rgba(255, 255, 255, .9)",
      width: 10,
      lineDash: [14, 14],
      lineDashOffset: 14,
      lineCap: "butt",
    }),
  }),
]

/** Props for the TrackMap component. */
interface TrackMapProps {
  /** Array of track point objects in WGS84. */
  points: TrackPoint[]
  /** Shared hover store for cross-component synchronization. */
  hoverStore: HoverStore
  /** Stroke color for the track line and hover marker. */
  color: string
  /** Optional CSS class for the outer container, overriding the default height. */
  className?: string
  /** Optional road closures to render on the map. */
  closures?: RoadClosure[]
  /** Resolved tile layer to display as the map background. */
  layer: MapLayer
}

/**
 * Renders an interactive map with the track line and hover marker.
 * Uses SwissTopo WMTS tiles for Swiss tracks or PMTiles vector tiles otherwise,
 * as determined by the layer prop.
 */
export default memo(function TrackMap({
  points,
  hoverStore,
  color,
  className,
  closures,
  layer,
}: TrackMapProps) {
  const mapRef = useRef<HTMLDivElement>(null)
  const tooltipRef = useRef<HTMLDivElement>(null)
  const mapInstance = useRef<OlMap | null>(null)
  const coordsRef = useRef<number[][]>([])
  const markerFeature = useRef<Feature | null>(null)
  const markerSource = useRef<VectorSource | null>(null)
  const closureLayerRef = useRef<VectorLayer | null>(null)
  const overlayRef = useRef<Overlay | null>(null)
  const [tooltip, setTooltip] = useState<RoadClosure | null>(null)
  const [tileErrorUrl, setTileErrorUrl] = useState<string | null>(null)
  const tileError = layer.type === "pmtiles" && tileErrorUrl === layer.url
  const [darkMode, setDarkMode] = useState(
    () => window.matchMedia("(prefers-color-scheme: dark)").matches
  )

  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = (e: MediaQueryListEvent) => setDarkMode(e.matches)
    mq.addEventListener("change", handler)
    return () => mq.removeEventListener("change", handler)
  }, [])

  const markerVisibleStyle = useMemo(
    () =>
      new Style({
        image: new Circle({
          radius: 6,
          fill: new Fill({ color }),
          stroke: new Stroke({ color: "#ffffff", width: 2 }),
        }),
      }),
    [color]
  )

  useEffect(() => {
    if (!mapRef.current || points.length === 0) return

    const coords = points.map((p) => projectPoint(p.lon, p.lat, layer.type))
    coordsRef.current = coords

    // Split track polyline into separate halo and inner features so the halo
    // can be drawn under the chevron halos and the colored inner over them.
    const trackHaloFeature = new Feature({
      geometry: new LineString(coords),
    })
    trackHaloFeature.setStyle(
      new Style({
        stroke: new Stroke({ color: "#ffffff", width: TRACK_LINE_HALO_WIDTH }),
      })
    )
    const trackInnerFeature = new Feature({
      geometry: new LineString(coords),
    })
    trackInnerFeature.setStyle(
      new Style({
        stroke: new Stroke({ color, width: TRACK_LINE_INNER_WIDTH }),
      })
    )

    const startFeature = new Feature({ geometry: new Point(coords[0]) })
    startFeature.setStyle(startStyle)
    const endFeature = new Feature({
      geometry: new Point(coords[coords.length - 1]),
    })
    endFeature.setStyle(endStyle)

    const trackHaloSource = new VectorSource({ features: [trackHaloFeature] })
    const trackInnerSource = new VectorSource({ features: [trackInnerFeature] })
    const endpointSource = new VectorSource({
      features: [startFeature, endFeature],
    })

    const marker = new Feature({ geometry: new Point(coords[0]) })
    marker.setStyle(markerHiddenStyle)
    markerFeature.current = marker

    const mSource = new VectorSource({ features: [marker] })
    markerSource.current = mSource

    const mapLayers = []

    if (layer.type === "swisstopo") {
      const tileGrid = getLV95TileGrid()
      mapLayers.push(
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
      pmtilesSource.on("error", (event) => {
        console.error("PMTiles source error:", event)
        setTileErrorUrl(layer.url)
      })
      mapLayers.push(
        new VectorTileLayer({
          source: pmtilesSource,
          style: createPmtilesStyleFn(darkMode),
        })
      )
    }

    const closureSource = new VectorSource()
    const closureLayer = new VectorLayer({ source: closureSource })
    closureLayerRef.current = closureLayer

    // Render chevron sprites at device pixel resolution and shrink them back
    // via Icon `scale: 1/dpr` so they're as crisp as the OL-drawn track line.
    const dpr = window.devicePixelRatio || 1
    const arrowIconScale = 1 / dpr
    const arrowHaloImg = buildArrowHaloCanvas(dpr)
    const arrowInnerImg = buildArrowInnerCanvas(color, dpr)
    const arrowHaloSource = new VectorSource()
    const arrowInnerSource = new VectorSource()

    const overlay = new Overlay({
      element: tooltipRef.current!,
      positioning: "bottom-center",
      offset: [0, -10],
      stopEvent: false,
    })
    overlayRef.current = overlay

    // Layer order: all white halos first (track + chevrons) so they compose
    // into one silhouette, then all colored inners on top, then closures and
    // endpoint markers above the line, hover marker on top.
    mapLayers.push(
      new VectorLayer({ source: trackHaloSource }),
      new VectorLayer({ source: arrowHaloSource }),
      new VectorLayer({ source: trackInnerSource }),
      new VectorLayer({ source: arrowInnerSource }),
      closureLayer,
      new VectorLayer({ source: endpointSource }),
      new VectorLayer({ source: mSource })
    )

    const view =
      layer.type === "swisstopo"
        ? new OlView(getLV95ViewConfig())
        : new OlView({})

    const map = new OlMap({
      target: mapRef.current,
      layers: mapLayers,
      overlays: [overlay],
      view,
    })

    const extent = trackHaloSource.getExtent()
    if (extent && !extentIsEmpty(extent)) {
      map.getView().fit(extent, { padding: [40, 40, 40, 40], maxZoom: 16 })
    }

    mapInstance.current = map

    const rebuildArrows = () => {
      arrowHaloSource.clear()
      arrowInnerSource.clear()
      const cs = coordsRef.current
      if (cs.length < 2) return
      const resolution = map.getView().getResolution()
      if (!resolution || !Number.isFinite(resolution)) return
      const spacing = resolution * ARROW_SCREEN_SPACING_PX

      // Cumulative segment lengths in projected map units.
      const cum: number[] = new Array(cs.length)
      cum[0] = 0
      for (let i = 1; i < cs.length; i++) {
        const dx = cs[i][0] - cs[i - 1][0]
        const dy = cs[i][1] - cs[i - 1][1]
        cum[i] = cum[i - 1] + Math.hypot(dx, dy)
      }
      const total = cum[cs.length - 1]
      // Skip when the visible track is too short to fit even one arrow comfortably.
      if (total < spacing * 1.2) return

      // Keep arrows away from the start/end markers.
      const margin = Math.min(spacing * 0.6, total * 0.1)
      const haloFeatures: Feature[] = []
      const innerFeatures: Feature[] = []
      let segIdx = 1
      for (let target = margin; target < total - margin; target += spacing) {
        while (segIdx < cum.length && cum[segIdx] < target) segIdx++
        if (segIdx >= cum.length) break
        const a = cs[segIdx - 1]
        const b = cs[segIdx]
        const segLen = cum[segIdx] - cum[segIdx - 1]
        const t = segLen > 0 ? (target - cum[segIdx - 1]) / segLen : 0
        const dx = b[0] - a[0]
        const dy = b[1] - a[1]
        if (dx === 0 && dy === 0) continue
        const point: [number, number] = [a[0] + t * dx, a[1] + t * dy]
        // OL Icon rotation is clockwise from the icon's natural orientation
        // (canvas top = north on an unrotated view); atan2(dx, dy) gives the
        // clockwise angle from north for a (dx, dy) step in projected coords.
        const rotation = Math.atan2(dx, dy)
        const haloFeature = new Feature({ geometry: new Point(point) })
        haloFeature.setStyle(
          new Style({
            image: new Icon({
              img: arrowHaloImg,
              anchor: [0.5, 0.5],
              rotation,
              rotateWithView: true,
              scale: arrowIconScale,
            }),
          })
        )
        const innerFeature = new Feature({ geometry: new Point(point) })
        innerFeature.setStyle(
          new Style({
            image: new Icon({
              img: arrowInnerImg,
              anchor: [0.5, 0.5],
              rotation,
              rotateWithView: true,
              scale: arrowIconScale,
            }),
          })
        )
        haloFeatures.push(haloFeature)
        innerFeatures.push(innerFeature)
      }
      arrowHaloSource.addFeatures(haloFeatures)
      arrowInnerSource.addFeatures(innerFeatures)
    }

    rebuildArrows()
    map.on("moveend", rebuildArrows)

    return () => {
      map.un("moveend", rebuildArrows)
      map.setTarget(undefined)
      mapInstance.current = null
      markerFeature.current = null
      markerSource.current = null
      closureLayerRef.current = null
      overlayRef.current = null
    }
  }, [points, color, markerVisibleStyle, layer, darkMode])

  // Find nearest track point within a pixel threshold of the pointer.
  const findNearest = useCallback((pixel: number[]) => {
    const map = mapInstance.current
    if (!map) return null
    const coord = map.getCoordinateFromPixel(pixel)
    if (!coord) return null

    const resolution = map.getView().getResolution() ?? 1
    const maxDist = 20 * resolution // 20px tolerance in map units.
    const maxDistSq = maxDist * maxDist

    const coords = coordsRef.current
    let bestIdx = -1
    let bestDist = Infinity
    for (let i = 0; i < coords.length; i++) {
      const dx = coords[i][0] - coord[0]
      const dy = coords[i][1] - coord[1]
      const dist = dx * dx + dy * dy
      if (dist < bestDist) {
        bestDist = dist
        bestIdx = i
      }
    }
    return bestDist <= maxDistSq ? bestIdx : null
  }, [])

  // Pointer move handler.
  useEffect(() => {
    const map = mapInstance.current
    if (!map) return

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onPointerMove = (evt: any) => {
      // OL fires pointermove during touch drags too; ignore those so a tap
      // (handled below via "click") drives the marker on touch devices.
      const orig = evt.originalEvent as PointerEvent | undefined
      if (orig && orig.pointerType && orig.pointerType !== "mouse") return
      const idx = findNearest(evt.pixel as number[])
      hoverStore.set(idx)
    }

    const onPointerLeave = (evt: PointerEvent) => {
      // Touch lock persists; only mouse leaving clears the marker.
      if (evt.pointerType && evt.pointerType !== "mouse") return
      hoverStore.set(null)
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onClick = (evt: any) => {
      // For touch/pen, a tap locks (or clears, when tapping empty) the marker.
      const orig = evt.originalEvent as PointerEvent | undefined
      if (orig && orig.pointerType === "mouse") return
      hoverStore.set(findNearest(evt.pixel as number[]))
    }

    map.on("pointermove" as never, onPointerMove)
    map.on("click", onClick)
    const viewport = map.getViewport()
    viewport.addEventListener("pointerleave", onPointerLeave)

    return () => {
      map.un("pointermove" as never, onPointerMove)
      map.un("click", onClick)
      viewport.removeEventListener("pointerleave", onPointerLeave)
    }
  }, [points, findNearest, hoverStore])

  // Update marker position imperatively when hover index changes.
  useEffect(() => {
    return hoverStore.subscribe(() => {
      const marker = markerFeature.current
      if (!marker) return

      const coords = coordsRef.current
      const idx = hoverStore.get()
      if (idx != null && idx >= 0 && idx < coords.length) {
        const geom = marker.getGeometry()
        if (!(geom instanceof Point)) return
        geom.setCoordinates(coords[idx])
        marker.setStyle(markerVisibleStyle)
      } else {
        marker.setStyle(markerHiddenStyle)
      }
    })
  }, [hoverStore, markerVisibleStyle])

  // Render road closure geometries on the map. `points` is in deps because
  // the main map-setup effect rebuilds the closure layer when points change;
  // we must re-add the features into that fresh layer.
  useEffect(() => {
    const closureLayer = closureLayerRef.current
    if (!closureLayer) return
    const source = closureLayer.getSource()!
    source.clear()

    if (!closures || closures.length === 0) return

    const fmt = new GeoJSON({
      dataProjection: "EPSG:4326",
      featureProjection: layer.type === "swisstopo" ? "EPSG:2056" : "EPSG:3857",
    })
    for (const c of closures) {
      if (c.type === "detour") continue
      try {
        const geom = fmt.readGeometry(JSON.parse(c.geometry))
        const feature = new Feature({ geometry: geom })
        feature.set("closure", c, true)
        feature.setStyle(
          c.type === "obstruction" ? detourStyle : closureStyle
        )
        source.addFeature(feature)
      } catch (err) {
        console.error(`Failed to parse closure geometry (${c.uuid}):`, err)
      }
    }
  }, [closures, layer, points])

  // Closure hover: show tooltip and highlight on pointer move over closure features.
  useEffect(() => {
    const map = mapInstance.current
    if (!map) return

    let highlightedFeature: Feature | null = null

    /** Highlights the closure feature under `pixel` and shows its tooltip. */
    const showAt = (pixel: number[]): boolean => {
      const overlay = overlayRef.current
      if (!overlay) return false

      // Restore previous highlight.
      if (highlightedFeature) {
        const prev = highlightedFeature.get("closure") as
          | RoadClosure
          | undefined
        highlightedFeature.setStyle(
          prev?.type === "obstruction" ? detourStyle : closureStyle
        )
        highlightedFeature = null
      }

      let found = false
      map.forEachFeatureAtPixel(
        pixel,
        (f) => {
          const c = (f as Feature).get("closure") as RoadClosure | undefined
          if (!c) return false
          found = true
          const feat = f as Feature
          feat.setStyle(
            c.type === "obstruction" ? detourStyleHover : closureStyleHover
          )
          highlightedFeature = feat
          overlay.setPosition(map.getCoordinateFromPixel(pixel)!)
          setTooltip(c)
          return true // stop iterating
        },
        { hitTolerance: 4 }
      )

      if (!found) {
        overlay.setPosition(undefined)
        setTooltip(null)
      }
      return found
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onMove = (evt: any) => {
      // Skip touch/pen; those drive the tooltip via tap (onClick) instead.
      const orig = evt.originalEvent as PointerEvent | undefined
      if (orig && orig.pointerType && orig.pointerType !== "mouse") return
      showAt(evt.pixel as number[])
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onClick = (evt: any) => {
      const orig = evt.originalEvent as PointerEvent | undefined
      if (orig && orig.pointerType === "mouse") return
      showAt(evt.pixel as number[])
    }

    const onLeave = (evt: PointerEvent) => {
      if (evt.pointerType && evt.pointerType !== "mouse") return
      if (highlightedFeature) {
        const prev = highlightedFeature.get("closure") as
          | RoadClosure
          | undefined
        highlightedFeature.setStyle(
          prev?.type === "obstruction" ? detourStyle : closureStyle
        )
        highlightedFeature = null
      }
      overlayRef.current?.setPosition(undefined)
      setTooltip(null)
    }

    map.on("pointermove" as never, onMove)
    map.on("click", onClick)
    const viewport = map.getViewport()
    viewport.addEventListener("pointerleave", onLeave)

    return () => {
      map.un("pointermove" as never, onMove)
      map.un("click", onClick)
      viewport.removeEventListener("pointerleave", onLeave)
    }
  }, [closures])

  const isNone = layer.type === "none"

  return (
    <div
      className={
        className ?? "relative h-[400px] w-full rounded-lg border border-border"
      }
    >
      {isNone && (
        <div
          className="absolute inset-0 rounded-lg"
          style={{ background: darkMode ? "#24201a" : "#f0ece4" }}
        />
      )}
      <div ref={mapRef} className="h-full w-full" />
      {tileError && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <span className="rounded bg-panel/90 px-3 py-1.5 text-sm text-error ring-1 ring-border">
            Map tiles unavailable
          </span>
        </div>
      )}
      <div
        ref={tooltipRef}
        className={tooltip ? "pointer-events-none" : "hidden"}
      >
        {tooltip && (
          <div className="max-w-xs rounded bg-panel px-2.5 py-1.5 text-xs shadow-md ring-1 ring-border">
            <p className="font-medium text-text">{tooltip.title}</p>
            {tooltip.reason && (
              <p className="mt-0.5 text-text-secondary">{tooltip.reason}</p>
            )}
            {(tooltip.startsAt || tooltip.endsAt) && (
              <p className="mt-0.5 text-text-muted">
                {tooltip.startsAt ? fmtDate(tooltip.startsAt) : "?"}
                {" – "}
                {tooltip.endsAt ? fmtDate(tooltip.endsAt) : "?"}
              </p>
            )}
            {tooltip.description && (
              <p className="mt-0.5 text-text-muted">{tooltip.description}</p>
            )}
            {tooltip.attribution.text && (
              <p className="mt-0.5 text-text-muted/70">
                Source: {tooltip.attribution.text}
              </p>
            )}
          </div>
        )}
      </div>
      <MapAttribution layer={layer} />
    </div>
  )
})
