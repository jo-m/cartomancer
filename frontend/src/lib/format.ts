/** Formats a distance in meters as km with one decimal. */
export function formatDistance(m: number): string {
  return `${(m / 1000).toFixed(1)} km`
}

/** Formats an ascent in meters as a rounded integer. */
export function formatAscent(m: number): string {
  return `${Math.round(m)} m`
}
