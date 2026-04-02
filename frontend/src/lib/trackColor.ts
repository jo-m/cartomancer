/**
 * getTrackColor reads the --color-track CSS custom property from the root element.
 * Used by components (TrackMap, ElevationProfile) that need a resolved hex value.
 */
export function getTrackColor(): string {
  return getComputedStyle(document.documentElement)
    .getPropertyValue("--color-track")
    .trim()
}
