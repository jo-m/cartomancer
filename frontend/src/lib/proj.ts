import { register } from "ol/proj/proj4"
import { get as getProjection, fromLonLat } from "ol/proj"
import proj4 from "proj4"

proj4.defs(
  "EPSG:2056",
  "+proj=somerc +lat_0=46.9524055555556 +lon_0=7.43958333333333 +k_0=1 +x_0=2600000 +y_0=1200000 +ellps=bessel +towgs84=674.374,15.056,405.346,0,0,0,0 +units=m +no_defs +type=crs"
)
register(proj4)

/** The Swiss LV95 projection (EPSG:2056), registered with OpenLayers. */
export const lv95 = getProjection("EPSG:2056")!

/**
 * Projects a WGS84 lon/lat point into the projection used by the given map
 * layer type. SwissTopo uses LV95 (EPSG:2056); all other layers use Web
 * Mercator (EPSG:3857). The result is a coordinate suitable for OpenLayers
 * geometries.
 */
export function projectPoint(
  lon: number,
  lat: number,
  layerType: "swisstopo" | "pmtiles" | "none"
): number[] {
  if (layerType === "swisstopo") {
    return proj4("EPSG:4326", "EPSG:2056", [lon, lat])
  }
  return fromLonLat([lon, lat])
}

export { proj4 }
