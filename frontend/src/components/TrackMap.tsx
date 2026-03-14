import { useEffect, useRef } from "react"
import OlMap from "ol/Map"
import OlView from "ol/View"
import TileLayer from "ol/layer/Tile"
import VectorLayer from "ol/layer/Vector"
import VectorSource from "ol/source/Vector"
import WMTS from "ol/source/WMTS"
import Feature from "ol/Feature"
import { LineString } from "ol/geom"
import { register } from "ol/proj/proj4"
import { get as getProjection } from "ol/proj"
import proj4 from "proj4"
import { Stroke, Style } from "ol/style"
import { getLV95TileGrid, getLV95ViewConfig } from "@swissgeo/coordinates/ol"

import "ol/ol.css"

proj4.defs(
  "EPSG:2056",
  "+proj=somerc +lat_0=46.9524055555556 +lon_0=7.43958333333333 +k_0=1 +x_0=2600000 +y_0=1200000 +ellps=bessel +towgs84=674.374,15.056,405.346,0,0,0,0 +units=m +no_defs +type=crs"
)
register(proj4)

const lv95 = getProjection("EPSG:2056")!

/** Props for the TrackMap component. */
interface TrackMapProps {
  /** Array of [lat, lon] coordinate pairs in WGS84. */
  points: [number, number][]
}

/** Renders an interactive swisstopo map with the track line. */
export default function TrackMap({ points }: TrackMapProps) {
  const mapRef = useRef<HTMLDivElement>(null)
  const mapInstance = useRef<OlMap | null>(null)

  useEffect(() => {
    if (!mapRef.current || points.length === 0) return

    const coords = points.map(([lat, lon]) =>
      proj4("EPSG:4326", "EPSG:2056", [lon, lat])
    )

    const trackFeature = new Feature({
      geometry: new LineString(coords),
    })
    trackFeature.setStyle(
      new Style({
        stroke: new Stroke({ color: "#e11d48", width: 3 }),
      })
    )

    const vectorSource = new VectorSource({ features: [trackFeature] })

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

    const viewConfig = getLV95ViewConfig()
    const map = new OlMap({
      target: mapRef.current,
      layers: [tileLayer, vectorLayer],
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
    }
  }, [points])

  return (
    <div
      ref={mapRef}
      className="h-[400px] w-full rounded-lg border border-gray-200"
    />
  )
}
