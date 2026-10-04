/** A track point with its cumulative distance in meters. */
export interface LitPolylinePoint {
  lat: number
  lon: number
  d: number
}

/**
 * Slices the polyline to the part between the cumulative distances startM and
 * endM. The range is clamped to the polyline, boundaries inside a segment are
 * interpolated, and interior vertices are kept so switchbacks stay faithful
 * to the drawn line. Returns [lon, lat] pairs for OpenLayers, or an empty
 * array for degenerate ranges and polylines with fewer than two points.
 */
export function slicePolylineByDistance(
  points: LitPolylinePoint[],
  startM: number,
  endM: number
): [number, number][] {
  if (points.length < 2 || endM <= startM) return []

  const first = points[0].d
  const last = points[points.length - 1].d
  const start = Math.max(startM, first)
  const end = Math.min(endM, last)
  if (end <= start) return []

  const result: [number, number][] = []

  const push = (p: [number, number]) => {
    const prev = result[result.length - 1]
    if (!prev || prev[0] !== p[0] || prev[1] !== p[1]) result.push(p)
  }

  /** Interpolates the coordinate at cumulative distance d within a segment. */
  const interp = (
    a: LitPolylinePoint,
    b: LitPolylinePoint,
    d: number
  ): [number, number] => {
    const span = b.d - a.d
    const t = span > 0 ? (d - a.d) / span : 0
    return [a.lon + (b.lon - a.lon) * t, a.lat + (b.lat - a.lat) * t]
  }

  for (let i = 0; i < points.length - 1; i++) {
    const a = points[i]
    const b = points[i + 1]
    if (b.d <= start || a.d >= end) continue

    const from = Math.max(start, a.d)
    const to = Math.min(end, b.d)
    push(interp(a, b, from))

    if (to < b.d) {
      // The range ends inside this segment; the start of the range was
      // already emitted, so the boundary closes the slice.
      push(interp(a, b, to))
      break
    }
    // The range continues past this segment: keep the segment's end vertex.
    push([b.lon, b.lat])
  }

  return result
}
