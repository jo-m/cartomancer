import { useCallback, useMemo } from "react"
import {
  boolParam,
  enumParam,
  numArrayParam,
  numberParam,
  rangeParam,
  strArrayParam,
  stringParam,
  useUrlState,
} from "./useUrlState"
import type { ParamDef } from "./useUrlState"
import type { LiveFilters } from "../components/TrackFilters"
import {
  DEFAULT_START_NEAR_RADIUS_M,
  START_NEAR_RADII_M,
} from "../types/trackGrid"
import type { SortBy, SortOrder, ViewMode } from "../types/trackGrid"

/** Default page size for the track grid. Also used to decide pager visibility. */
export const DEFAULT_PAGE_SIZE = 24

/**
 * URL parameter definition for the "start near" filter. Serialized as
 * "lat,lon,radiusM" with six decimals on lat/lon (~10 cm); null when
 * absent. Two-part legacy values default to DEFAULT_START_NEAR_RADIUS_M.
 * Any radius outside the allowed set is snapped to the default.
 */
function nearParam(): ParamDef<{
  lat: number
  lon: number
  radiusM: number
} | null> {
  return {
    defaultValue: null,
    parse: (raw) => {
      const parts = raw.split(",").map(Number)
      if (parts.length < 2 || !parts.every(Number.isFinite)) return null
      const [lat, lon] = parts
      if (lat < -90 || lat > 90 || lon < -180 || lon > 180) return null
      const rawRadius = parts[2]
      const radiusM = (START_NEAR_RADII_M as readonly number[]).includes(
        rawRadius
      )
        ? rawRadius
        : DEFAULT_START_NEAR_RADIUS_M
      return { lat, lon, radiusM }
    },
    serialize: (value) =>
      value
        ? `${value.lat.toFixed(6)},${value.lon.toFixed(6)},${value.radiusM}`
        : "",
    equals: (a, b) => {
      if (a === null && b === null) return true
      if (a === null || b === null) return false
      return a.lat === b.lat && a.lon === b.lon && a.radiusM === b.radiusM
    },
  }
}

const urlSchema = {
  q: stringParam(),
  dist: rangeParam(),
  ascent: rangeParam(),
  vis: enumParam("all" as const, ["all", "public", "private"] as const),
  type: enumParam("all" as const, ["all", "recorded", "planned"] as const),
  starred: boolParam(),
  sports: numArrayParam(),
  subsports: numArrayParam(),
  tags: strArrayParam(),
  tagsAnd: boolParam(),
  sort: enumParam(
    "created_at" as SortBy,
    ["created_at", "total_distance_m", "total_ascent_m"] as const
  ),
  order: enumParam("desc" as SortOrder, ["asc", "desc"] as const),
  page: numberParam(1),
  pageSize: numberParam(DEFAULT_PAGE_SIZE),
  view: enumParam("list" as ViewMode, ["list", "map"] as const),
  near: nearParam(),
}

/** Plain URL-state value type derived from urlSchema. */
type UrlState = {
  q: string
  dist: [number, number] | null
  ascent: [number, number] | null
  vis: "all" | "public" | "private"
  type: "all" | "recorded" | "planned"
  starred: boolean
  sports: number[]
  subsports: number[]
  tags: string[]
  tagsAnd: boolean
  sort: SortBy
  order: SortOrder
  page: number
  pageSize: number
  view: ViewMode
  near: { lat: number; lon: number; radiusM: number } | null
}

/** Converts URL state to LiveFilters. */
function urlToFilters(url: UrlState): LiveFilters {
  return {
    search: url.q,
    distRange: url.dist,
    ascentRange: url.ascent,
    visibility: url.vis,
    trackType: url.type,
    onlyStarred: url.starred,
    sports: url.sports,
    subSports: url.subsports,
    tags: url.tags,
    tagsAnd: url.tagsAnd,
    sortBy: url.sort,
    sortOrder: url.order,
    startNear: url.near,
  }
}

/** Converts LiveFilters into a partial URL-state update. */
function filtersToUrl(f: LiveFilters): Partial<UrlState> {
  return {
    q: f.search,
    dist: f.distRange,
    ascent: f.ascentRange,
    vis: f.visibility,
    type: f.trackType,
    starred: f.onlyStarred,
    sports: f.sports,
    subsports: f.subSports,
    tags: f.tags,
    tagsAnd: f.tagsAnd,
    sort: f.sortBy,
    order: f.sortOrder,
    near: f.startNear,
  }
}

/**
 * Syncs the track grid URL search params and exposes them in a form
 * convenient for the grid.
 *
 * @returns
 *   - `urlState`: raw URL-backed values (use for `page`, `pageSize`, `view`).
 *   - `setUrlState`: partial-update setter for any URL field.
 *   - `applied`: URL state mapped into the {@link LiveFilters} shape used by
 *     the filter panel and the query builder.
 *   - `commitFilters(live)`: writes a full {@link LiveFilters} object back to
 *     the URL and resets `page` to 1. Use after debouncing user input.
 */
export function useTrackGridUrl() {
  const schema = useMemo(() => urlSchema, [])
  const [urlState, setUrlState] = useUrlState(schema)
  const applied = useMemo(() => urlToFilters(urlState as UrlState), [urlState])
  const commitFilters = useCallback(
    (live: LiveFilters) => {
      setUrlState({ ...filtersToUrl(live), page: 1 })
    },
    [setUrlState]
  )
  return { urlState, setUrlState, applied, commitFilters }
}
