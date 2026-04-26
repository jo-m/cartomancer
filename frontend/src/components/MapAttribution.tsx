import type { MapLayer } from "../lib/mapLayer"

interface MapAttributionProps {
  layer: MapLayer
}

/**
 * Renders a small attribution overlay anchored to the bottom-right corner of
 * a map container. The displayed sources depend on which basemap is active:
 * SwissTopo for the Swiss WMTS layer, Protomaps + OpenStreetMap for PMTiles
 * vector tiles. Renders nothing when the resolved layer has no basemap.
 *
 * The parent must be `position: relative` for the overlay to position
 * correctly.
 */
export default function MapAttribution({ layer }: MapAttributionProps) {
  if (layer.type === "none") return null
  return (
    <div className="pointer-events-none absolute bottom-0 right-0 z-10 bg-panel/80 px-1.5 py-0.5 text-xs text-text-muted">
      {layer.type === "swisstopo" ? (
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
      ) : (
        <>
          Map data:&nbsp;
          <a
            href="https://protomaps.com"
            target="_blank"
            rel="noopener noreferrer"
            className="pointer-events-auto hover:underline"
          >
            Protomaps
          </a>
          {" \u00a9 "}
          <a
            href="https://openstreetmap.org/copyright"
            target="_blank"
            rel="noopener noreferrer"
            className="pointer-events-auto hover:underline"
          >
            OpenStreetMap
          </a>
        </>
      )}
    </div>
  )
}
