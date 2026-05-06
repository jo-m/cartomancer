/** Formats a duration in milliseconds as "Xh YYmin". */
export function fmtElapsed(ms: number): string {
  const totalMin = Math.round(ms / 60000)
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  if (h === 0) return `${m}min`
  return `${h}h ${m.toString().padStart(2, "0")}min`
}

/**
 * Formats a timestamp as HH:MM in 24-hour format. Accepts either an epoch
 * millisecond value or an ISO 8601 string.
 */
export function fmtClock(value: number | string): string {
  const ts = typeof value === "string" ? new Date(value).getTime() : value
  return new Date(ts).toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}

/** Formats an ISO timestamp as a short calendar date, e.g. "5 Jan 2025". */
export function fmtDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  })
}

/** Formats an ISO timestamp as a date with HH:MM, e.g. "5 Jan 2025, 14:30". */
export function fmtDateTime(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}

/**
 * Formats an ISO timestamp as a long absolute string suitable for tooltips,
 * including seconds and the user's timezone abbreviation.
 */
export function fmtAbsolute(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    timeZoneName: "short",
    hour12: false,
  })
}

const RTF = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" })

/**
 * Formats an ISO timestamp as a coarse relative phrase like "3 min ago" or
 * "in 2 hours". Falls back to fmtDate for offsets beyond one week.
 */
export function fmtRelative(iso: string): string {
  const deltaMs = new Date(iso).getTime() - Date.now()
  const absSec = Math.abs(deltaMs) / 1000
  if (absSec < 45) return RTF.format(Math.round(deltaMs / 1000), "second")
  if (absSec < 60 * 60)
    return RTF.format(Math.round(deltaMs / 60_000), "minute")
  if (absSec < 24 * 60 * 60)
    return RTF.format(Math.round(deltaMs / 3_600_000), "hour")
  if (absSec < 7 * 24 * 60 * 60)
    return RTF.format(Math.round(deltaMs / 86_400_000), "day")
  return fmtDate(iso)
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
