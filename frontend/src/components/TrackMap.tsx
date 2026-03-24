import { useEffect, useRef, useCallback, useMemo, useState, memo } from "react"
import OlMap from "ol/Map"
import OlView from "ol/View"
import Overlay from "ol/Overlay"
import TileLayer from "ol/layer/Tile"
import VectorLayer from "ol/layer/Vector"
import VectorSource from "ol/source/Vector"
import WMTS from "ol/source/WMTS"
import Feature from "ol/Feature"
import GeoJSON from "ol/format/GeoJSON"
import { LineString, Point } from "ol/geom"
import { register } from "ol/proj/proj4"
import { get as getProjection } from "ol/proj"
import proj4 from "proj4"
import { Circle, Fill, Icon, Stroke, Style } from "ol/style"
import { getLV95TileGrid, getLV95ViewConfig } from "@swissgeo/coordinates/ol"

import type { HoverStore } from "../hooks/useHoverSync"

import "ol/ol.css"

proj4.defs(
  "EPSG:2056",
  "+proj=somerc +lat_0=46.9524055555556 +lon_0=7.43958333333333 +k_0=1 +x_0=2600000 +y_0=1200000 +ellps=bessel +towgs84=674.374,15.056,405.346,0,0,0,0 +units=m +no_defs +type=crs"
)
register(proj4)

const lv95 = getProjection("EPSG:2056")!

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

const geoJSONFormat = new GeoJSON({
  dataProjection: "EPSG:4326",
  featureProjection: "EPSG:2056",
})

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
}

/** Renders an interactive swisstopo map with the track line and hover marker. */
export default memo(function TrackMap({
  points,
  hoverStore,
  color,
  className,
  closures,
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

    const coords = points.map((p) =>
      proj4("EPSG:4326", "EPSG:2056", [p.lon, p.lat])
    )
    coordsRef.current = coords

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

    const tileGrid = getLV95TileGrid()
    const tileLayer = new TileLayer({
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

    const vectorLayer = new VectorLayer({ source: vectorSource })
    const markerLayer = new VectorLayer({ source: mSource })

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

    const viewConfig = getLV95ViewConfig()
    const map = new OlMap({
      target: mapRef.current,
      layers: [tileLayer, vectorLayer, closureLayer, markerLayer],
      overlays: [overlay],
      view: new OlView(viewConfig),
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
  }, [points, color])

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
    const layer = closureLayerRef.current
    if (!layer) return
    const source = layer.getSource()!
    source.clear()

    if (!closures || closures.length === 0) return

    for (const c of closures) {
      try {
        const geom = geoJSONFormat.readGeometry(JSON.parse(c.geometry))
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

  return (
    <div
      className={
        className ??
        "relative h-[400px] w-full rounded-lg border border-gray-200"
      }
    >
      <div ref={mapRef} className="h-full w-full" />
      <div
        ref={tooltipRef}
        className={tooltip ? "pointer-events-none" : "hidden"}
      >
        {tooltip && (
          <div className="max-w-xs rounded bg-white px-2.5 py-1.5 text-xs shadow-md ring-1 ring-gray-200">
            <p className="font-medium text-gray-900">{tooltip.title}</p>
            {tooltip.reason && (
              <p className="mt-0.5 text-gray-600">{tooltip.reason}</p>
            )}
            {(tooltip.startsAt || tooltip.endsAt) && (
              <p className="mt-0.5 text-gray-500">
                {tooltip.startsAt ?? "?"} &ndash; {tooltip.endsAt ?? "?"}
              </p>
            )}
            {tooltip.description && (
              <p className="mt-0.5 text-gray-500">{tooltip.description}</p>
            )}
            {tooltip.attribution.text && (
              <p className="mt-0.5 text-gray-400">
                Source: {tooltip.attribution.text}
              </p>
            )}
          </div>
        )}
      </div>
      <div className="pointer-events-none absolute bottom-0 right-0 z-10 px-1.5 py-0.5 text-xs text-gray-600 bg-white/80">
        Map data:&nbsp;
        <a
          href="https://www.swisstopo.admin.ch/"
          target="_blank"
          rel="noopener noreferrer"
          className="pointer-events-auto hover:underline"
        >
          SwissTopo
        </a>
      </div>
    </div>
  )
})
