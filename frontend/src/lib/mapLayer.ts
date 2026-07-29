/** Geographic bounding box in WGS84. */
export interface Bbox {
  minLat: number
  maxLat: number
  minLon: number
  maxLon: number
}

/**
 * The bounding box of the SwissTopo coverage area with zoom level 5.
 * Tracks whose bbox falls within this area use the SwissTopo WMTS layer.
 */
export const SWISSTOPO_BBOX: Bbox = {
  minLat: 45.33,
  maxLat: 48.18,
  minLon: 5.4,
  maxLon: 11.19,
}

/** A resolved tile layer selection for display in the track map. */
export type MapLayer =
  { type: "swisstopo" } | { type: "pmtiles"; url: string } | { type: "none" }

/** Subset of the /api/maps response items needed for layer selection. */
interface MapBuildInfo {
  uuid: string
  maxZoom?: number | null
  bboxMinLon?: number | null
  bboxMinLat?: number | null
  bboxMaxLon?: number | null
  bboxMaxLat?: number | null
  /** ISO 8601 datetime when the build was uploaded. */
  uploaded?: string | null
}

/**
 * Returns the area of the build's declared bbox, or Infinity when no bbox
 * is set (treated as global coverage and therefore lowest detail among ties).
 */
function buildBboxArea(m: MapBuildInfo): number {
  if (
    m.bboxMinLon == null ||
    m.bboxMinLat == null ||
    m.bboxMaxLon == null ||
    m.bboxMaxLat == null
  ) {
    return Infinity
  }
  return (m.bboxMaxLon - m.bboxMinLon) * (m.bboxMaxLat - m.bboxMinLat)
}

/** Returns true if outer fully contains inner. */
function bboxContains(outer: Bbox, inner: Bbox): boolean {
  return (
    inner.minLat >= outer.minLat &&
    inner.maxLat <= outer.maxLat &&
    inner.minLon >= outer.minLon &&
    inner.maxLon <= outer.maxLon
  )
}

/**
 * Computes the bounding box of an array of track points.
 *
 * @param points - Array of objects with lat/lon in WGS84.
 * @returns The bounding box, or null for an empty array.
 */
export function computeTrackBbox(
  points: { lat: number; lon: number }[]
): Bbox | null {
  if (points.length === 0) return null
  let minLat = Infinity,
    maxLat = -Infinity,
    minLon = Infinity,
    maxLon = -Infinity
  for (const p of points) {
    if (p.lat < minLat) minLat = p.lat
    if (p.lat > maxLat) maxLat = p.lat
    if (p.lon < minLon) minLon = p.lon
    if (p.lon > maxLon) maxLon = p.lon
  }
  return { minLat, maxLat, minLon, maxLon }
}

/**
 * Returns the union of two bounding boxes. A null input is treated as an
 * empty bbox so unionBbox(null, x) === x.
 */
export function unionBbox(a: Bbox | null, b: Bbox | null): Bbox | null {
  if (!a) return b
  if (!b) return a
  return {
    minLat: Math.min(a.minLat, b.minLat),
    maxLat: Math.max(a.maxLat, b.maxLat),
    minLon: Math.min(a.minLon, b.minLon),
    maxLon: Math.max(a.maxLon, b.maxLon),
  }
}

/**
 * Selects the appropriate tile layer for a given track bounding box.
 *
 * Returns "swisstopo" if the track lies entirely within the SwissTopo
 * coverage area. Otherwise returns the highest-zoom PMTiles map build
 * whose bbox contains the track bbox (a build with no bbox is treated as
 * global coverage). Returns "none" when no suitable PMTiles build exists.
 *
 * @param trackBbox - The bounding box of the track in WGS84.
 * @param maps - Available ready map builds from GET /api/maps.
 */
export function selectMapLayer(
  trackBbox: Bbox,
  maps: MapBuildInfo[]
): MapLayer {
  if (bboxContains(SWISSTOPO_BBOX, trackBbox)) {
    return { type: "swisstopo" }
  }

  const covering = maps.filter((m) => {
    const hasBbox =
      m.bboxMinLon != null &&
      m.bboxMinLat != null &&
      m.bboxMaxLon != null &&
      m.bboxMaxLat != null
    if (!hasBbox) return true
    return bboxContains(
      {
        minLat: m.bboxMinLat!,
        maxLat: m.bboxMaxLat!,
        minLon: m.bboxMinLon!,
        maxLon: m.bboxMaxLon!,
      },
      trackBbox
    )
  })

  if (covering.length === 0) return { type: "none" }

  // Pick the build with the highest maxZoom. Break ties by smaller bbox area
  // (more detail per tile), then by newer upload time, then by uuid so the
  // result is fully deterministic regardless of input order or sort stability.
  const best = [...covering].sort((a, b) => {
    const zoomDiff = (b.maxZoom ?? 0) - (a.maxZoom ?? 0)
    if (zoomDiff !== 0) return zoomDiff
    const areaDiff = buildBboxArea(a) - buildBboxArea(b)
    if (areaDiff !== 0) return areaDiff
    const uploadedA = a.uploaded ?? ""
    const uploadedB = b.uploaded ?? ""
    if (uploadedA !== uploadedB) return uploadedB.localeCompare(uploadedA)
    return a.uuid.localeCompare(b.uuid)
  })[0]
  return { type: "pmtiles", url: `/api/maps/${best.uuid}` }
}
