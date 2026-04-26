/**
 * getTrackColor reads the --color-track CSS custom property from the root element.
 * Used by components (TrackMap, ElevationProfile) that need a resolved hex value.
 */
export function getTrackColor(): string {
  return getComputedStyle(document.documentElement)
    .getPropertyValue("--color-track")
    .trim()
}

/**
 * Hashes a string to an unsigned 32-bit integer using the FNV-1a algorithm.
 * Stable across runs and deterministic for a given input.
 */
function hashFnv1a(s: string): number {
  let h = 0x811c9dc5
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  return h >>> 0
}

/**
 * trackColorFromUUID returns a stable, vibrant CSS color string for a track
 * derived deterministically from its UUID. The hash drives the HSL hue
 * channel; saturation and lightness are fixed to keep all tracks readable
 * against both light and dark map backgrounds. Used by the multi-track
 * map view to give every track a distinguishable color.
 */
export function trackColorFromUUID(uuid: string): string {
  const hue = hashFnv1a(uuid) % 360
  return `hsl(${hue}, 80%, 45%)`
}
