import { useEffect, useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { $api } from "../api/client"
import OlMap from "ol/Map"
import OlView from "ol/View"
import TileLayer from "ol/layer/Tile"
import VectorLayer from "ol/layer/Vector"
import VectorSource from "ol/source/Vector"
import WMTS from "ol/source/WMTS"
import Feature from "ol/Feature"
import { LineString, Point as OlPoint } from "ol/geom"
import { Circle, Fill, Stroke, Style } from "ol/style"
import { getLV95TileGrid, getLV95ViewConfig } from "@swissgeo/coordinates/ol"
import PageContainer from "../components/ui/PageContainer"
import { lv95, proj4 } from "../lib/proj"

import "ol/ol.css"

const segmentStyle = [
  new Style({ stroke: new Stroke({ color: "#ffffff", width: 5 }) }),
  new Style({ stroke: new Stroke({ color: "#3b82f6", width: 3 }) }),
]

const segmentHoverStyle = [
  new Style({ stroke: new Stroke({ color: "#ffffff", width: 7 }) }),
  new Style({ stroke: new Stroke({ color: "#f59e0b", width: 4 }) }),
]

const junctionStyle = new Style({
  image: new Circle({
    radius: 5,
    fill: new Fill({ color: "#ef4444" }),
    stroke: new Stroke({ color: "#ffffff", width: 1.5 }),
  }),
})

/** Segments displays all extracted segments and junctions on a single map. */
export default function Segments() {
  const navigate = useNavigate()
  const mapRef = useRef<HTMLDivElement>(null)
  const mapInstanceRef = useRef<OlMap | null>(null)
  const [hoveredUuid, setHoveredUuid] = useState<string | null>(null)

  const {
    data: segData,
    isLoading: segLoading,
    error: segError,
  } = $api.useQuery("get", "/admin/segments")
  const { data: juncData } = $api.useQuery("get", "/admin/segments/junctions")

  useEffect(() => {
    if (!mapRef.current || !segData) return

    const segmentSource = new VectorSource()
    const junctionSource = new VectorSource()

    for (const seg of segData.segments) {
      try {
        const points: [number, number][] = JSON.parse(seg.polyline)
        const coords = points.map(([lat, lon]) =>
          proj4("EPSG:4326", "EPSG:2056", [lon, lat])
        )
        if (coords.length < 2) continue

        const feature = new Feature({ geometry: new LineString(coords) })
        feature.setId(seg.uuid)
        feature.set("segmentUuid", seg.uuid)
        feature.set(
          "label",
          `${(seg.distanceM / 1000).toFixed(1)} km / ${seg.nTracks} tracks`
        )
        feature.setStyle(segmentStyle)
        segmentSource.addFeature(feature)
      } catch {
        // Skip malformed polylines.
      }
    }

    if (juncData) {
      for (const j of juncData.junctions) {
        const coord = proj4("EPSG:4326", "EPSG:2056", [j.lon, j.lat])
        const feature = new Feature({ geometry: new OlPoint(coord) })
        feature.setStyle(junctionStyle)
        junctionSource.addFeature(feature)
      }
    }

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

    const segmentLayer = new VectorLayer({ source: segmentSource })
    const junctionLayer = new VectorLayer({ source: junctionSource })

    const map = new OlMap({
      target: mapRef.current,
      layers: [tileLayer, segmentLayer, junctionLayer],
      view: new OlView(getLV95ViewConfig()),
    })

    const extent = segmentSource.getExtent()
    if (extent && extent[0] !== Infinity) {
      map.getView().fit(extent, { padding: [40, 40, 40, 40], maxZoom: 14 })
    }

    let currentHover: Feature | null = null
    map.on("pointermove", (e) => {
      if (currentHover) {
        currentHover.setStyle(segmentStyle)
        currentHover = null
        setHoveredUuid(null)
      }

      map.forEachFeatureAtPixel(e.pixel, (f) => {
        const feature = f as Feature
        if (feature.get("segmentUuid")) {
          feature.setStyle(segmentHoverStyle)
          currentHover = feature
          setHoveredUuid(feature.get("segmentUuid") as string)
        }
        return true
      })

      map.getTargetElement().style.cursor = currentHover ? "pointer" : ""
    })

    map.on("click", (e) => {
      map.forEachFeatureAtPixel(e.pixel, (f) => {
        const feature = f as Feature
        const uuid = feature.get("segmentUuid") as string | undefined
        if (uuid) {
          navigate(`/admin/segments/${uuid}`)
          return true
        }
        return false
      })
    })

    mapInstanceRef.current = map

    return () => {
      map.setTarget(undefined)
      mapInstanceRef.current = null
    }
  }, [segData, juncData, navigate])

  const hoveredSeg = hoveredUuid
    ? segData?.segments.find((s) => s.uuid === hoveredUuid)
    : null

  return (
    <PageContainer size="xl" className="max-w-7xl">
      <h1 className="text-lg font-semibold text-text">Segments</h1>
      <p className="mt-1 text-sm text-text-muted">
        Road/way segments shared by multiple tracks. Click a segment to see
        details.
      </p>

      {segLoading && <p className="mt-6 text-sm text-text-muted">Loading...</p>}
      {segError && (
        <p role="alert" className="mt-6 text-sm text-error">
          {segError.message}
        </p>
      )}

      {segData && (
        <>
          <p className="mt-2 text-xs text-text-muted">
            {segData.segments.length} segments
            {juncData ? `, ${juncData.junctions.length} junctions` : ""}
          </p>

          <div className="relative mt-3">
            <div
              ref={mapRef}
              className="h-[calc(100vh-220px)] w-full rounded-lg border border-border"
            />

            {hoveredSeg && (
              <div className="absolute bottom-4 left-4 rounded-lg border border-border bg-panel/90 px-3 py-2 text-sm shadow-sm">
                <p className="font-medium text-text">
                  {(hoveredSeg.distanceM / 1000).toFixed(1)} km
                </p>
                <p className="text-xs text-text-muted">
                  {hoveredSeg.nTracks} tracks
                </p>
              </div>
            )}
          </div>
        </>
      )}
    </PageContainer>
  )
}
