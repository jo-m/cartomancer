import { useEffect, useRef, useCallback, useMemo, memo } from "react"
import OlMap from "ol/Map"
import OlView from "ol/View"
import TileLayer from "ol/layer/Tile"
import VectorLayer from "ol/layer/Vector"
import VectorSource from "ol/source/Vector"
import WMTS from "ol/source/WMTS"
import Feature from "ol/Feature"
import { LineString, Point } from "ol/geom"
import { register } from "ol/proj/proj4"
import { get as getProjection } from "ol/proj"
import proj4 from "proj4"
import { Circle, Fill, Stroke, Style } from "ol/style"
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

interface TrackPoint {
  lat: number
  lon: number
  ele: number
  d: number
}

/** Props for the TrackMap component. */
interface TrackMapProps {
  /** Array of track point objects in WGS84. */
  points: TrackPoint[]
  /** Shared hover store for cross-component synchronization. */
  hoverStore: HoverStore
  /** Stroke color for the track line and hover marker. */
  color: string
}

/** Renders an interactive swisstopo map with the track line and hover marker. */
export default memo(function TrackMap({
  points,
  hoverStore,
  color,
}: TrackMapProps) {
  const mapRef = useRef<HTMLDivElement>(null)
  const mapInstance = useRef<OlMap | null>(null)
  const coordsRef = useRef<number[][]>([])
  const markerFeature = useRef<Feature | null>(null)
  const markerSource = useRef<VectorSource | null>(null)

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
    trackFeature.setStyle(
      new Style({
        stroke: new Stroke({ color, width: 4 }),
      })
    )

    const vectorSource = new VectorSource({ features: [trackFeature] })

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

    const viewConfig = getLV95ViewConfig()
    const map = new OlMap({
      target: mapRef.current,
      layers: [tileLayer, vectorLayer, markerLayer],
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
    }
  }, [points, color])

  // Find nearest track point to a map coordinate.
  const findNearest = useCallback((pixel: number[]) => {
    const map = mapInstance.current
    if (!map) return null
    const coord = map.getCoordinateFromPixel(pixel)
    if (!coord) return null

    const coords = coordsRef.current
    let bestIdx = 0
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
    return bestIdx
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

  return (
    <div
      ref={mapRef}
      className="h-[400px] w-full rounded-lg border border-gray-200"
    />
  )
})
