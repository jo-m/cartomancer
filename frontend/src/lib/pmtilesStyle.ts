import { Fill, Stroke, Style, Text } from "ol/style"
import type { FeatureLike } from "ol/Feature"

type StyleFn = (feature: FeatureLike) => Style | Style[] | void

/**
 * Creates an OpenLayers style function for Protomaps Basemap v4 vector tiles.
 *
 * Static styles are pre-allocated per call and reused across all tile renders.
 * Place labels create a new Style per feature (text varies), but OL caches
 * these per tile.
 *
 * @param dark - When true, returns styles suited for a dark colour scheme.
 */
export function createPmtilesStyleFn(dark: boolean): StyleFn {
  // --- land / water / vegetation ---
  const earthStyle = new Style({
    fill: new Fill({ color: dark ? "#24201a" : "#f0ece4" }),
  })
  const waterStyle = new Style({
    fill: new Fill({ color: dark ? "#1e2c38" : "#ccdded" }),
  })
  const parkStyle = new Style({
    fill: new Fill({ color: dark ? "#1e2c1e" : "#d8e8d8" }),
  })
  const buildingStyle = new Style({
    fill: new Fill({ color: dark ? "#2e2820" : "#e6e2d8" }),
  })

  // --- roads ---
  const highwayStyles = [
    new Style({
      stroke: new Stroke({ color: dark ? "#3a3228" : "#ffffff", width: 4 }),
    }),
    new Style({
      stroke: new Stroke({ color: dark ? "#4a4238" : "#c8c0b0", width: 2.5 }),
    }),
  ]
  const majorStyles = [
    new Style({
      stroke: new Stroke({ color: dark ? "#3a3228" : "#ffffff", width: 3 }),
    }),
    new Style({
      stroke: new Stroke({ color: dark ? "#423c30" : "#d0cab8", width: 1.8 }),
    }),
  ]
  const mediumStyle = new Style({
    stroke: new Stroke({ color: dark ? "#302c24" : "#d8d4c8", width: 1.2 }),
  })
  const minorStyle = new Style({
    stroke: new Stroke({ color: dark ? "#2a2620" : "#e0dcd4", width: 0.7 }),
  })

  // --- boundaries ---
  const countryBorderStyle = new Style({
    stroke: new Stroke({
      color: dark ? "#6a5e50" : "#9a8e80",
      width: 1.5,
      lineDash: [5, 4],
    }),
  })
  const regionBorderStyle = new Style({
    stroke: new Stroke({
      color: dark ? "#48403a" : "#bcb4a8",
      width: 1,
      lineDash: [3, 5],
    }),
  })
  const countyBorderStyle = new Style({
    stroke: new Stroke({
      color: dark ? "#38322c" : "#cec8be",
      width: 0.75,
      lineDash: [2, 6],
    }),
  })

  // --- place label colours ---
  const labelFill = new Fill({ color: dark ? "#e0d4c0" : "#2c1c0c" })
  const labelHalo = new Stroke({
    color: dark ? "#18140f" : "#f5efe8",
    width: 3,
  })

  return (feature: FeatureLike): Style | Style[] | void => {
    const layer = feature.get("layer") as string
    const kind = feature.get("kind") as string

    switch (layer) {
      case "earth":
        return earthStyle

      case "water":
        return waterStyle

      // landcover: broad natural coverage (forest, scrub, glacier, grassland…)
      case "landcover":
        if (kind === "forest" || kind === "scrub" || kind === "grassland")
          return parkStyle
        if (kind === "glacier")
          return new Style({
            fill: new Fill({ color: dark ? "#2a3840" : "#e8f4f8" }),
          })
        return

      // landuse: mapped human/natural areas (parks, farmland…)
      case "landuse":
        if (["park", "urban_green", "forest", "scrub", "grass"].includes(kind))
          return parkStyle
        return

      case "roads":
        if (kind === "highway") return highwayStyles
        if (kind === "major_road") return majorStyles
        if (kind === "medium_road") return mediumStyle
        return minorStyle

      case "buildings":
        return buildingStyle

      case "boundaries": {
        if (kind === "country") return countryBorderStyle
        if (kind === "region") return regionBorderStyle
        if (kind === "county") return countyBorderStyle
        return
      }

      case "places": {
        const name = feature.get("name") as string | undefined
        if (!name) return
        const rank = (feature.get("population_rank") as number) ?? 0
        const fontSize = rank >= 13 ? 14 : rank >= 9 ? 12 : rank >= 5 ? 11 : 10
        const fontWeight = rank >= 9 ? "bold" : "normal"
        return new Style({
          text: new Text({
            text: name,
            font: `${fontWeight} ${fontSize}px sans-serif`,
            fill: labelFill,
            stroke: labelHalo,
            overflow: true,
          }),
        })
      }

      default:
        return
    }
  }
}
