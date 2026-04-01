import { useEffect, useRef } from "react"
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

import "ol/ol.css"

proj4.defs(
  "EPSG:2056",
  "+proj=somerc +lat_0=46.9524055555556 +lon_0=7.43958333333333 +k_0=1 +x_0=2600000 +y_0=1200000 +ellps=bessel +towgs84=674.374,15.056,405.346,0,0,0,0 +units=m +no_defs +type=crs"
)
register(proj4)

const lv95 = getProjection("EPSG:2056")!

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
  /** JSON-encoded array of [lat, lon] pairs. */
  polyline: string
}

/** SegmentMap renders a segment polyline on a SwissTopo map. */
export default function SegmentMap({ polyline }: SegmentMapProps) {
  const mapRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!mapRef.current) return

    let coords: number[][]
    try {
      const points: [number, number][] = JSON.parse(polyline)
      coords = points.map(([lat, lon]) =>
        proj4("EPSG:4326", "EPSG:2056", [lon, lat])
      )
    } catch {
      return
    }

    if (coords.length < 2) return

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
    if (extent && extent[0] !== Infinity) {
      map.getView().fit(extent, { padding: [40, 40, 40, 40], maxZoom: 16 })
    }

    return () => {
      map.setTarget(undefined)
    }
  }, [polyline])

  return (
    <div ref={mapRef} className="h-80 w-full rounded-lg border border-border" />
  )
}
