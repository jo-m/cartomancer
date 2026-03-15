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

/** Interpolates forecast timestamps to cover every track point index. */
export function buildForecastTimes(
  forecastPoints: { index: number; time: string }[],
  numTrackPoints: number
): number[] {
  const result = new Array<number>(numTrackPoints)
  if (forecastPoints.length === 0) return result.fill(0)

  const fp = forecastPoints.map((p) => ({
    i: p.index,
    ts: new Date(p.time).getTime(),
  }))

  for (let i = 0; i < numTrackPoints; i++) {
    if (i <= fp[0].i) {
      result[i] = fp[0].ts
    } else if (i >= fp[fp.length - 1].i) {
      result[i] = fp[fp.length - 1].ts
    } else {
      let j = 0
      while (j < fp.length - 1 && fp[j + 1].i <= i) j++
      const t = (i - fp[j].i) / (fp[j + 1].i - fp[j].i)
      result[i] = Math.round(fp[j].ts + t * (fp[j + 1].ts - fp[j].ts))
    }
  }
  return result
}
