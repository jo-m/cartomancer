import { Fill, Stroke, Style } from "ol/style"
import type { FeatureLike } from "ol/Feature"

type StyleFn = (feature: FeatureLike) => Style | Style[] | void

/**
 * Creates an OpenLayers style function for Protomaps Basemap v4 vector tiles.
 *
 * Styles are pre-allocated for the given mode and reused across all tile
 * renders; the returned function must not be called with a different dark
 * value after creation.
 *
 * @param dark - When true, returns styles suited for a dark colour scheme.
 */
export function createPmtilesStyleFn(dark: boolean): StyleFn {
  if (dark) {
    return buildStyleFn({
      earth: "#24201a",
      water: "#1e2c38",
      park: "#1e2c1e",
      building: "#2e2820",
      highwayCasing: "#3a3228",
      highwayFill: "#4a4238",
      majorCasing: "#3a3228",
      majorFill: "#423c30",
      medium: "#302c24",
      minor: "#2a2620",
    })
  }
  return buildStyleFn({
    earth: "#f0ece4",
    water: "#ccdded",
    park: "#d8e8d8",
    building: "#e6e2d8",
    highwayCasing: "#ffffff",
    highwayFill: "#c8c0b0",
    majorCasing: "#ffffff",
    majorFill: "#d0cab8",
    medium: "#d8d4c8",
    minor: "#e0dcd4",
  })
}

interface Palette {
  earth: string
  water: string
  park: string
  building: string
  highwayCasing: string
  highwayFill: string
  majorCasing: string
  majorFill: string
  medium: string
  minor: string
}

function buildStyleFn(p: Palette): StyleFn {
  const earthStyle = new Style({ fill: new Fill({ color: p.earth }) })
  const waterStyle = new Style({ fill: new Fill({ color: p.water }) })
  const parkStyle = new Style({ fill: new Fill({ color: p.park }) })
  const buildingStyle = new Style({ fill: new Fill({ color: p.building }) })
  const highwayCasing = new Style({
    stroke: new Stroke({ color: p.highwayCasing, width: 4 }),
  })
  const highwayFill = new Style({
    stroke: new Stroke({ color: p.highwayFill, width: 2.5 }),
  })
  const majorCasing = new Style({
    stroke: new Stroke({ color: p.majorCasing, width: 3 }),
  })
  const majorFill = new Style({
    stroke: new Stroke({ color: p.majorFill, width: 1.8 }),
  })
  const mediumStyle = new Style({
    stroke: new Stroke({ color: p.medium, width: 1.2 }),
  })
  const minorStyle = new Style({
    stroke: new Stroke({ color: p.minor, width: 0.7 }),
  })
  const highwayStyles = [highwayCasing, highwayFill]
  const majorStyles = [majorCasing, majorFill]

  return (feature: FeatureLike): Style | Style[] | void => {
    const layer = feature.get("layer") as string
    const kind = feature.get("kind") as string
    switch (layer) {
      case "earth":
        return earthStyle
      case "natural":
        if (kind === "water") return waterStyle
        return
      case "landuse":
        if (
          ["park", "urban_green", "forest", "scrub", "grass"].includes(kind)
        ) {
          return parkStyle
        }
        return
      case "water":
        return waterStyle
      case "roads":
        if (kind === "highway") return highwayStyles
        if (kind === "major_road") return majorStyles
        if (kind === "medium_road") return mediumStyle
        return minorStyle
      case "buildings":
        return buildingStyle
      default:
        return
    }
  }
}
