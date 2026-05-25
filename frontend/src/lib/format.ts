/** Formats a distance in meters as km with one decimal. */
export function formatDistance(m: number): string {
  return `${(m / 1000).toFixed(1)} km`
}

/** Formats an ascent in meters as a rounded integer. */
export function formatAscent(m: number): string {
  return `${Math.round(m)} m`
}

/**
 * Formats an erythemal UV dose in Standard Erythema Doses. Returns "low" for
 * doses below 0.5 SED (well below the ~2 SED minimal erythemal threshold for
 * type-II skin); otherwise shows one decimal up to 9.9 SED and a rounded
 * integer at or above 10 SED.
 */
export function formatUVDoseSED(sed: number): string {
  if (sed < 0.5) {
    return "low"
  }
  if (sed < 10) {
    return `${sed.toFixed(1)} SED`
  }
  return `${Math.round(sed)} SED`
}
