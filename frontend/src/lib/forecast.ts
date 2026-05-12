const WIND_DIRS = ["N", "NE", "E", "SE", "S", "SW", "W", "NW"] as const

/** Returns a cardinal direction label for a meteorological wind direction in degrees. */
export function windDirLabel(deg: number): string {
  const idx = Math.round(deg / 45) % 8
  return WIND_DIRS[idx]
}

const REL_WIND_LABELS = [
  "Headwind",
  "Head-right",
  "Crosswind R",
  "Tail-right",
  "Tailwind",
  "Tail-left",
  "Crosswind L",
  "Head-left",
] as const

/** Returns a human label for a relative wind direction (0 = headwind, 180 = tailwind). */
export function relWindLabel(deg: number): string {
  const idx = Math.round(deg / 45) % 8
  return REL_WIND_LABELS[idx]
}

/** Computes the headwind component (positive = headwind, negative = tailwind). */
export function headwindComponent(
  windSpeedMs: number,
  relativeWindDeg: number
): number {
  return windSpeedMs * Math.cos((relativeWindDeg * Math.PI) / 180)
}

/** Returns a fill color for a relative wind sector (headwind=red, tailwind=green, cross=gray). */
export function relativeSectorColor(i: number): string {
  if (i === 0 || i === 1 || i === 7) return "#ef4444"
  if (i === 3 || i === 4 || i === 5) return "#10b981"
  return "#9ca3af"
}

/** Returns a uniform fill color for compass wind sectors. */
export function compassSectorColor(): string {
  return "#6366f1"
}
