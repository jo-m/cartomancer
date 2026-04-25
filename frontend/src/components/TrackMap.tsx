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
import { fromLonLat } from "ol/proj"
import { LineString, Point } from "ol/geom"
import { Circle, Fill, Stroke, Style, Icon } from "ol/style"
import { getLV95TileGrid, getLV95ViewConfig } from "@swissgeo/coordinates/ol"
import { lv95, proj4 } from "../lib/proj"
import { PMTilesVectorSource } from "ol-pmtiles"

import type { MapLayer } from "../lib/mapLayer"
import { createPmtilesStyleFn } from "../lib/pmtilesStyle"
import type { HoverStore } from "../hooks/useHoverSync"

import "ol/ol.css"

const markerHiddenStyle = new Style({})

const startStyle = new Style({
  image: new Circle({
    radius: 7,
    fill: new Fill({ color: "#22c55e" }),
    stroke: new Stroke({ color: "#ffffff", width: 2 }),
  }),
})

/** Builds a checkerboard circle icon rendered on a canvas. */
function buildEndStyleCanvas(): HTMLCanvasElement {
  const size = 18
  const canvas = document.createElement("canvas")
  canvas.width = size
  canvas.height = size
  const ctx = canvas.getContext("2d")!
  const cx = size / 2
  const r = size / 2 - 1

  // White base circle.
  ctx.beginPath()
  ctx.arc(cx, cx, r, 0, Math.PI * 2)
  ctx.fillStyle = "#ffffff"
  ctx.fill()

  // Clip to inner circle, draw 4x4 checkerboard.
  ctx.save()
  ctx.beginPath()
  ctx.arc(cx, cx, r - 1, 0, Math.PI * 2)
  ctx.clip()
  const cell = (size - 4) / 4
  for (let row = 0; row < 4; row++) {
    for (let col = 0; col < 4; col++) {
      ctx.fillStyle = (row + col) % 2 === 0 ? "#1f2937" : "#ffffff"
      ctx.fillRect(2 + col * cell, 2 + row * cell, cell, cell)
    }
  }
  ctx.restore()

  // White border ring.
  ctx.beginPath()
  ctx.arc(cx, cx, r, 0, Math.PI * 2)
  ctx.strokeStyle = "#ffffff"
  ctx.lineWidth = 2
  ctx.stroke()

  return canvas
}

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

/** A road closure to display on the map. */
export interface RoadClosure {
  uuid: string
  type: string
  title: string
  startsAt?: string | null
  endsAt?: string | null
  reason?: string | null
  description?: string | null
  geometry: string
  attribution: { text: string; href: string }
}

const detourStyle = new Style({
  stroke: new Stroke({ color: "rgba(245, 158, 11, 0.7)", width: 10 }),
})

const detourStyleHover = new Style({
  stroke: new Stroke({ color: "rgba(245, 158, 11, 0.9)", width: 10 }),
})

const closureStyle = [
  new Style({
    stroke: new Stroke({
      color: "rgba(220, 38, 38, 0.7)",
      width: 10,
      lineDash: [14, 14],
      lineCap: "butt",
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

/** Projects a WGS84 lon/lat point into the map projection for the given layer type. */
function projectPoint(
  lon: number,
  lat: number,
  layerType: MapLayer["type"]
): number[] {
  if (layerType === "swisstopo") {
    return proj4("EPSG:4326", "EPSG:2056", [lon, lat])
  }
  return fromLonLat([lon, lat])
}

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
  // GeoJSON format is projection-dependent; updated when the map is initialized.
  const geoJSONFormatRef = useRef<GeoJSON>(
    new GeoJSON({ dataProjection: "EPSG:4326", featureProjection: "EPSG:3857" })
  )
  const [tooltip, setTooltip] = useState<RoadClosure | null>(null)
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

    geoJSONFormatRef.current = new GeoJSON({
      dataProjection: "EPSG:4326",
      featureProjection: layer.type === "swisstopo" ? "EPSG:2056" : "EPSG:3857",
    })

    const trackFeature = new Feature({
      geometry: new LineString(coords),
    })
    trackFeature.setStyle([
      new Style({ stroke: new Stroke({ color: "#ffffff", width: 7 }) }),
      new Style({ stroke: new Stroke({ color, width: 4 }) }),
    ])

    const startFeature = new Feature({ geometry: new Point(coords[0]) })
    startFeature.setStyle(startStyle)
    const endFeature = new Feature({
      geometry: new Point(coords[coords.length - 1]),
    })
    endFeature.setStyle(endStyle)

    const vectorSource = new VectorSource({
      features: [trackFeature, startFeature, endFeature],
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
      mapLayers.push(
        new VectorTileLayer({
          source: new PMTilesVectorSource({ url: layer.url }),
          style: createPmtilesStyleFn(darkMode),
        })
      )
    }

    const closureSource = new VectorSource()
    const closureLayer = new VectorLayer({ source: closureSource })
    closureLayerRef.current = closureLayer

    const overlay = new Overlay({
      element: tooltipRef.current!,
      positioning: "bottom-center",
      offset: [0, -10],
      stopEvent: false,
    })
    overlayRef.current = overlay

    mapLayers.push(
      new VectorLayer({ source: vectorSource }),
      closureLayer,
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

    const extent = vectorSource.getExtent()
    if (extent && extent[0] !== Infinity) {
      map.getView().fit(extent, { padding: [40, 40, 40, 40], maxZoom: 16 })
    }

    mapInstance.current = map

    return () => {
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
      const idx = findNearest(evt.pixel as number[])
      hoverStore.set(idx)
    }

    const onPointerLeave = () => {
      hoverStore.set(null)
    }

    map.on("pointermove" as never, onPointerMove)
    const viewport = map.getViewport()
    viewport.addEventListener("pointerleave", onPointerLeave)

    return () => {
      map.un("pointermove" as never, onPointerMove)
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
        const geom = marker.getGeometry() as Point
        geom.setCoordinates(coords[idx])
        marker.setStyle(markerVisibleStyle)
      } else {
        marker.setStyle(markerHiddenStyle)
      }
    })
  }, [hoverStore, markerVisibleStyle])

  // Render road closure geometries on the map.
  useEffect(() => {
    const closureLayer = closureLayerRef.current
    if (!closureLayer) return
    const source = closureLayer.getSource()!
    source.clear()

    if (!closures || closures.length === 0) return

    const fmt = geoJSONFormatRef.current
    for (const c of closures) {
      try {
        const geom = fmt.readGeometry(JSON.parse(c.geometry))
        const feature = new Feature({ geometry: geom })
        feature.set("closure", c, true)
        feature.setStyle(c.type === "detour" ? detourStyle : closureStyle)
        source.addFeature(feature)
      } catch {
        // Skip closures with unparseable geometry.
      }
    }
  }, [closures])

  // Closure hover: show tooltip and highlight on pointer move over closure features.
  useEffect(() => {
    const map = mapInstance.current
    if (!map) return

    let highlightedFeature: Feature | null = null

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onMove = (evt: any) => {
      const pixel = evt.pixel as number[]
      const overlay = overlayRef.current
      if (!overlay) return

      // Restore previous highlight.
      if (highlightedFeature) {
        const prev = highlightedFeature.get("closure") as
          | RoadClosure
          | undefined
        highlightedFeature.setStyle(
          prev?.type === "detour" ? detourStyle : closureStyle
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
            c.type === "detour" ? detourStyleHover : closureStyleHover
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
    }

    const onLeave = () => {
      if (highlightedFeature) {
        const prev = highlightedFeature.get("closure") as
          | RoadClosure
          | undefined
        highlightedFeature.setStyle(
          prev?.type === "detour" ? detourStyle : closureStyle
        )
        highlightedFeature = null
      }
      overlayRef.current?.setPosition(undefined)
      setTooltip(null)
    }

    map.on("pointermove" as never, onMove)
    const viewport = map.getViewport()
    viewport.addEventListener("pointerleave", onLeave)

    return () => {
      map.un("pointermove" as never, onMove)
      viewport.removeEventListener("pointerleave", onLeave)
    }
  }, [closures])

  const isNone = layer.type === "none"
  const isSwiss = layer.type === "swisstopo"

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
                {tooltip.startsAt ?? "?"} &ndash; {tooltip.endsAt ?? "?"}
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
      <div className="pointer-events-none absolute bottom-0 right-0 z-10 px-1.5 py-0.5 text-xs text-text-muted bg-panel/80">
        {isSwiss ? (
          <>
            Map data:&nbsp;
            <a
              href="https://www.swisstopo.admin.ch/"
              target="_blank"
              rel="noopener noreferrer"
              className="pointer-events-auto hover:underline"
            >
              SwissTopo
            </a>
          </>
        ) : !isNone ? (
          <>
            Map data:&nbsp;
            <a
              href="https://openstreetmap.org/copyright"
              target="_blank"
              rel="noopener noreferrer"
              className="pointer-events-auto hover:underline"
            >
              OpenStreetMap contributors
            </a>
            ,&nbsp;
            <a
              href="https://maps.protomaps.com/builds/"
              target="_blank"
              rel="noopener noreferrer"
              className="pointer-events-auto hover:underline"
            >
              Protomaps
            </a>
          </>
        ) : null}
      </div>
    </div>
  )
})
