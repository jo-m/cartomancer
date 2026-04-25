import { useEffect, useMemo, useRef } from "react"
import OlMap from "ol/Map"
import OlView from "ol/View"
import TileLayer from "ol/layer/Tile"
import VectorLayer from "ol/layer/Vector"
import VectorSource from "ol/source/Vector"
import WMTS from "ol/source/WMTS"
import Feature from "ol/Feature"
import { isEmpty as extentIsEmpty } from "ol/extent"
import { LineString, Point } from "ol/geom"
import { Circle, Fill, Stroke, Style } from "ol/style"
import { getLV95TileGrid, getLV95ViewConfig } from "@swissgeo/coordinates/ol"
import { lv95, proj4 } from "../lib/proj"

import "ol/ol.css"

const lineStyle = [
  new Style({ stroke: new Stroke({ color: "#ffffff", width: 7 }) }),
  new Style({ stroke: new Stroke({ color: "#3b82f6", width: 4 }) }),
]

const startStyle = new Style({
  image: new Circle({
    radius: 7,
    fill: new Fill({ color: "#22c55e" }),
    stroke: new Stroke({ color: "#ffffff", width: 2 }),
  }),
})

const endStyle = new Style({
  image: new Circle({
    radius: 7,
    fill: new Fill({ color: "#ef4444" }),
    stroke: new Stroke({ color: "#ffffff", width: 2 }),
  }),
})

interface SegmentMapProps {
  /** JSON-encoded array of [lat, lon] pairs (latitude first, as sent by the API). */
  polyline: string
}

/** SegmentMap renders a segment polyline on a SwissTopo map. */
export default function SegmentMap({ polyline }: SegmentMapProps) {
  const mapRef = useRef<HTMLDivElement>(null)

  const coords = useMemo<number[][] | null>(() => {
    try {
      const points: [number, number][] = JSON.parse(polyline)
      const result = points.map(([lat, lon]) =>
        proj4("EPSG:4326", "EPSG:2056", [lon, lat])
      )
      return result.length >= 2 ? result : null
    } catch {
      return null
    }
  }, [polyline])

  useEffect(() => {
    if (!mapRef.current || !coords) return

    const trackFeature = new Feature({ geometry: new LineString(coords) })
    trackFeature.setStyle(lineStyle)

    const startFeature = new Feature({ geometry: new Point(coords[0]) })
    startFeature.setStyle(startStyle)

    const endFeature = new Feature({
      geometry: new Point(coords[coords.length - 1]),
    })
    endFeature.setStyle(endStyle)

    const vectorSource = new VectorSource({
      features: [trackFeature, startFeature, endFeature],
    })

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

    const map = new OlMap({
      target: mapRef.current,
      layers: [tileLayer, vectorLayer],
      view: new OlView(getLV95ViewConfig()),
    })

    const extent = vectorSource.getExtent()
    if (extent && !extentIsEmpty(extent)) {
      map.getView().fit(extent, { padding: [40, 40, 40, 40], maxZoom: 16 })
    }

    return () => {
      map.setTarget(undefined)
    }
  }, [coords])

  if (!coords) {
    return (
      <div className="flex h-80 w-full items-center justify-center rounded-lg border border-border bg-panel text-sm text-text-muted">
        Map unavailable
      </div>
    )
  }

  return (
    <div ref={mapRef} className="h-80 w-full rounded-lg border border-border" />
  )
}
