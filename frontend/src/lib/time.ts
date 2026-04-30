/** Formats a duration in milliseconds as "Xh YYmin". */
export function fmtElapsed(ms: number): string {
  const totalMin = Math.round(ms / 60000)
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  if (h === 0) return `${m}min`
  return `${h}h ${m.toString().padStart(2, "0")}min`
}

/** Formats a timestamp as HH:MM in 24-hour format. */
export function fmtClock(ts: number): string {
  return new Date(ts).toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}

/**
 * Interpolates forecast timestamps onto every track point by cumulative
 * distance, so /points and /forecast can use independent point counts.
 *
 * `forecastPoints` carries the forecast samples with `distanceM` along the
 * track; `trackDistancesM` is the cumulative distance of each track point.
 * For points before the first forecast sample the first timestamp is used,
 * and for points beyond the last sample the last timestamp is used.
 */
export function buildForecastTimes(
  forecastPoints: { distanceM: number; time: string }[],
  trackDistancesM: number[]
): number[] {
  const result = new Array<number>(trackDistancesM.length)
  if (forecastPoints.length === 0) return result.fill(0)

  const fp = forecastPoints.map((p) => ({
    d: p.distanceM,
    ts: new Date(p.time).getTime(),
  }))

  let j = 0
  for (let i = 0; i < trackDistancesM.length; i++) {
    const d = trackDistancesM[i]
    if (d <= fp[0].d) {
      result[i] = fp[0].ts
      continue
    }
    if (d >= fp[fp.length - 1].d) {
      result[i] = fp[fp.length - 1].ts
      continue
    }
    while (j < fp.length - 1 && fp[j + 1].d <= d) j++
    const span = fp[j + 1].d - fp[j].d
    const t = span > 0 ? (d - fp[j].d) / span : 0
    result[i] = Math.round(fp[j].ts + t * (fp[j + 1].ts - fp[j].ts))
  }
  return result
}
