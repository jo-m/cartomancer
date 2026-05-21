/**
 * Allowed radius values (in meters) for the "start near" filter. The user
 * picks one of these in the map toolbar; the API uses the chosen value as
 * `startNearRadiusM`.
 */
export const START_NEAR_RADII_M = [100, 500, 1000, 5000, 10000] as const

/** Default radius used when a start-location pin is first dropped. */
export const DEFAULT_START_NEAR_RADIUS_M = 500

export type SortBy = "created_at" | "total_distance_m" | "total_ascent_m"
export type SortOrder = "asc" | "desc"
export type ViewMode = "list" | "map"
