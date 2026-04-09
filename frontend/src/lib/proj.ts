import { register } from "ol/proj/proj4"
import { get as getProjection } from "ol/proj"
import proj4 from "proj4"

proj4.defs(
  "EPSG:2056",
  "+proj=somerc +lat_0=46.9524055555556 +lon_0=7.43958333333333 +k_0=1 +x_0=2600000 +y_0=1200000 +ellps=bessel +towgs84=674.374,15.056,405.346,0,0,0,0 +units=m +no_defs +type=crs"
)
register(proj4)

/** The Swiss LV95 projection (EPSG:2056), registered with OpenLayers. */
export const lv95 = getProjection("EPSG:2056")!

export { proj4 }
