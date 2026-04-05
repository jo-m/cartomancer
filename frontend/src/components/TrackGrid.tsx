import { useState, useEffect, useRef, useMemo, useCallback } from "react"
import { Link } from "react-router-dom"
import { $api, fetchClient } from "../api/client"
import {
  useUrlState,
  stringParam,
  numberParam,
  boolParam,
  numArrayParam,
  strArrayParam,
  rangeParam,
  enumParam,
} from "../hooks/useUrlState"
import SvgPreview from "./SvgPreview"
import { useSession } from "../context/SessionContext"
import StarIcon from "../assets/StarIcon"
import SvgIcon from "../assets/SvgIcon"
import distanceSvg from "../assets/distance.svg?raw"
import elevationSvg from "../assets/elevation.svg?raw"
import temperatureSvg from "../assets/temperature.svg?raw"
import rainSvg from "../assets/rain.svg?raw"
import cardCornerSvg from "../assets/card-corner.svg?raw"
import TagsInput from "./TagsInput"
import Toast from "./Toast"
import Button from "./ui/Button"
import Select from "./ui/Select"
import ToggleGroup from "./ui/ToggleGroup"
import Input from "./ui/Input"
import SectionHeading from "./ui/SectionHeading"
import PageContainer from "./ui/PageContainer"
import {
  SPORT_LABELS,
  SUB_SPORT_LABELS,
  SUB_SPORTS_BY_SPORT,
} from "../lib/sports"
import { useQueryClient } from "@tanstack/react-query"

const DEFAULT_PAGE_SIZE = 24
const PAGE_SIZE_OPTIONS = [12, 24, 48, 96]

function formatDistance(m: number): string {
  return `${(m / 1000).toFixed(1)} km`
}

function formatAscent(m: number): string {
  return `${Math.round(m)} m`
}

/** MiniWindRose renders a tiny 4-sector wind rose as an inline SVG. */
function MiniWindRose({
  head,
  right,
  tail,
  left,
}: {
  head?: number
  right?: number
  tail?: number
  left?: number
}) {
  const vals = [head ?? 0, right ?? 0, tail ?? 0, left ?? 0]
  const maxVal = Math.max(...vals)
  if (maxVal < 0.1) return null
  const fracs = vals.map((v) => v / maxVal)

  const colors = ["#ef4444", "#9ca3af", "#10b981", "#9ca3af"]
  const angles = [-90, 0, 90, 180]
  const cx = 16
  const cy = 16
  const maxR = 14
  const wedge = 35

  return (
    <svg
      width="32"
      height="32"
      viewBox="0 0 32 32"
      className="inline-block shrink-0"
      aria-label="Wind rose"
    >
      {fracs.map((f, i) => {
        if (f < 0.05) return null
        const r = maxR * f
        const a = angles[i]
        const a1 = ((a - wedge) * Math.PI) / 180
        const a2 = ((a + wedge) * Math.PI) / 180
        const x1 = cx + r * Math.cos(a1)
        const y1 = cy + r * Math.sin(a1)
        const x2 = cx + r * Math.cos(a2)
        const y2 = cy + r * Math.sin(a2)
        return (
          <path
            key={i}
            d={`M${cx},${cy} L${x1},${y1} A${r},${r} 0 0,1 ${x2},${y2} Z`}
            fill={colors[i]}
            fillOpacity={0.6}
            stroke={colors[i]}
            strokeWidth={0.5}
          />
        )
      })}
      <circle cx={cx} cy={cy} r={1} fill="var(--color-text-muted)" />
    </svg>
  )
}

interface DualRangeSliderProps {
  absoluteMin: number
  absoluteMax: number
  valueMin: number
  valueMax: number
  step: number
  formatValue: (v: number) => string
  onChange: (min: number, max: number) => void
}

function DualRangeSlider({
  absoluteMin,
  absoluteMax,
  valueMin,
  valueMax,
  step,
  formatValue,
  onChange,
}: DualRangeSliderProps) {
  const outerRef = useRef<HTMLDivElement>(null)
  const activeThumb = useRef<"min" | "max" | null>(null)

  const range = absoluteMax - absoluteMin || 1

  function valueFromClientX(clientX: number): number {
    if (!outerRef.current) return absoluteMin
    const { left, width } = outerRef.current.getBoundingClientRect()
    const p = Math.max(0, Math.min(1, (clientX - left - 8) / (width - 16)))
    return Math.round((absoluteMin + p * range) / step) * step
  }

  function pickThumb(v: number): "min" | "max" {
    const dMin = Math.abs(v - valueMin)
    const dMax = Math.abs(v - valueMax)
    if (dMin !== dMax) return dMin < dMax ? "min" : "max"
    return v >= (absoluteMin + absoluteMax) / 2 ? "min" : "max"
  }

  function applyValue(v: number) {
    if (activeThumb.current === "min") onChange(Math.min(v, valueMax), valueMax)
    else if (activeThumb.current === "max")
      onChange(valueMin, Math.max(v, valueMin))
  }

  function onPointerDown(e: React.PointerEvent<HTMLDivElement>) {
    e.currentTarget.setPointerCapture(e.pointerId)
    const v = valueFromClientX(e.clientX)
    activeThumb.current = pickThumb(v)
    applyValue(v)
  }

  function onPointerMove(e: React.PointerEvent<HTMLDivElement>) {
    if (!e.currentTarget.hasPointerCapture(e.pointerId)) return
    applyValue(valueFromClientX(e.clientX))
  }

  function onPointerUp() {
    activeThumb.current = null
  }

  function thumbLeft(v: number): string {
    const frac = (v - absoluteMin) / range
    return `calc(${frac * 100}% + ${8 - frac * 16}px)`
  }

  const minFrac = (valueMin - absoluteMin) / range
  const maxFrac = (valueMax - absoluteMin) / range
  const highlightStyle = {
    left: `calc(${minFrac * 100}% + ${8 - minFrac * 16}px)`,
    right: `calc(${(1 - maxFrac) * 100}% + ${maxFrac * 16 - 8}px)`,
  }

  const minActive = valueMin > absoluteMin
  const maxActive = valueMax < absoluteMax

  return (
    <div>
      <div
        ref={outerRef}
        className="relative h-5 cursor-grab select-none active:cursor-grabbing"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        role="slider"
        aria-valuemin={absoluteMin}
        aria-valuemax={absoluteMax}
        aria-valuenow={valueMin}
        tabIndex={0}
      >
        <div className="absolute inset-x-2 top-1/2 h-1.5 -translate-y-1/2 rounded-full bg-slider-track" />
        <div
          className="absolute top-1/2 h-1.5 -translate-y-1/2 rounded-full bg-slider-fill"
          style={highlightStyle}
        />
        <div
          className="pointer-events-none absolute top-1/2 h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 bg-slider-thumb"
          style={{
            left: thumbLeft(valueMin),
            borderColor: minActive
              ? "var(--color-slider-thumb-active)"
              : "var(--color-slider-thumb-inactive)",
          }}
        />
        <div
          className="pointer-events-none absolute top-1/2 h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 bg-slider-thumb"
          style={{
            left: thumbLeft(valueMax),
            borderColor: maxActive
              ? "var(--color-slider-thumb-active)"
              : "var(--color-slider-thumb-inactive)",
          }}
        />
      </div>
      <div className="mt-1 flex justify-between text-xs">
        <span
          className={
            minActive ? "font-medium text-text-secondary" : "text-text-muted"
          }
        >
          {formatValue(valueMin)}
        </span>
        <span
          className={
            maxActive ? "font-medium text-text-secondary" : "text-text-muted"
          }
        >
          {formatValue(valueMax)}
        </span>
      </div>
    </div>
  )
}

type Range = [number, number]

const SPORT_IDS = [1, 2] as const

type SortBy = "created_at" | "total_distance_m" | "total_ascent_m"
type SortOrder = "asc" | "desc"

const SORT_OPTIONS: { value: SortBy; label: string }[] = [
  { value: "created_at", label: "Uploaded at" },
  { value: "total_distance_m", label: "Distance" },
  { value: "total_ascent_m", label: "Ascent" },
]

interface LiveFilters {
  search: string
  distRange: Range | null
  ascentRange: Range | null
  visibility: "all" | "public" | "private"
  trackType: "all" | "recorded" | "planned"
  onlyStarred: boolean
  sports: number[]
  subSports: number[]
  tags: string[]
  tagsAnd: boolean
  sortBy: SortBy
  sortOrder: SortOrder
}

export interface TrackGridProps {
  mode: "public" | "user"
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
}

/** Converts URL state to LiveFilters. */
function urlToFilters(url: {
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
}): LiveFilters {
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
  }
}

/** Converts LiveFilters to URL state params. */
function filtersToUrl(f: LiveFilters) {
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
  }
}

/** TrackGrid renders a filterable, paginated grid of track cards. */
export default function TrackGrid({ mode }: TrackGridProps) {
  const onlyMine = mode === "user"
  const { user } = useSession()
  const queryClient = useQueryClient()

  const { data: stats } = $api.useQuery("get", "/tracks/statistics", {
    params: { query: { onlyMine } },
  })

  const absMaxDistKm =
    stats?.totalDistanceM.max != null
      ? Math.ceil(stats.totalDistanceM.max / 1000)
      : 0
  const absMaxAscentM =
    stats?.totalAscentM.max != null
      ? Math.ceil(stats.totalAscentM.max / 10) * 10
      : 0

  const schema = useMemo(() => urlSchema, [])
  const [urlState, setUrlState] = useUrlState(schema)

  const applied = useMemo(() => urlToFilters(urlState), [urlState])
  const [live, setLive] = useState<LiveFilters>(() => applied)
  const page = urlState.page
  const pageSize = urlState.pageSize

  function setPage(update: number | ((prev: number) => number)) {
    const next = typeof update === "function" ? update(page) : update
    setUrlState({ page: next })
  }

  function setPageSize(size: number) {
    setUrlState({ pageSize: size, page: 1 })
  }

  const prevLiveRef = useRef(live)
  useEffect(() => {
    if (prevLiveRef.current === live) return
    prevLiveRef.current = live
    const timer = setTimeout(() => {
      setUrlState({ ...filtersToUrl(live), page: 1 })
    }, 200)
    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [live])

  const distMin = live.distRange?.[0] ?? 0
  const distMax = live.distRange?.[1] ?? absMaxDistKm
  const ascentMin = live.ascentRange?.[0] ?? 0
  const ascentMax = live.ascentRange?.[1] ?? absMaxAscentM

  function setDistRange(min: number, max: number) {
    const r: Range | null =
      min === 0 && max === absMaxDistKm ? null : [min, max]
    setLive((prev) => ({ ...prev, distRange: r }))
  }

  function setAscentRange(min: number, max: number) {
    const r: Range | null =
      min === 0 && max === absMaxAscentM ? null : [min, max]
    setLive((prev) => ({ ...prev, ascentRange: r }))
  }

  function toggleSport(id: number) {
    setLive((prev) => {
      const next = prev.sports.includes(id)
        ? prev.sports.filter((s) => s !== id)
        : [...prev.sports, id]
      const validSubSports = next.flatMap((s) => SUB_SPORTS_BY_SPORT[s] ?? [])
      return {
        ...prev,
        sports: next,
        subSports: prev.subSports.filter((ss) => validSubSports.includes(ss)),
      }
    })
  }

  function toggleSubSport(id: number) {
    setLive((prev) => ({
      ...prev,
      subSports: prev.subSports.includes(id)
        ? prev.subSports.filter((s) => s !== id)
        : [...prev.subSports, id],
    }))
  }

  const availableSubSports =
    live.sports.length === 0
      ? []
      : [
          ...new Set(
            live.sports.flatMap((s) =>
              (SUB_SPORTS_BY_SPORT[s] ?? []).filter((ss) => ss !== 0)
            )
          ),
        ]

  const appliedDistMin = applied.distRange?.[0] ?? 0
  const appliedDistMax = applied.distRange?.[1] ?? absMaxDistKm
  const appliedAscentMin = applied.ascentRange?.[0] ?? 0
  const appliedAscentMax = applied.ascentRange?.[1] ?? absMaxAscentM

  let publicParam: boolean | undefined
  if (mode === "public") {
    publicParam = true
  } else if (applied.visibility === "public") {
    publicParam = true
  } else if (applied.visibility === "private") {
    publicParam = false
  }

  const trackTypeParam: number[] | undefined =
    applied.trackType === "recorded"
      ? [2]
      : applied.trackType === "planned"
        ? [1]
        : undefined

  const starMutation = $api.useMutation("post", "/tracks/{uuid}/star")
  const unstarMutation = $api.useMutation("delete", "/tracks/{uuid}/star")

  async function toggleStar(
    e: React.MouseEvent,
    trackUuid: string,
    starred: boolean
  ) {
    e.preventDefault()
    e.stopPropagation()
    if (starred) {
      await unstarMutation.mutateAsync({
        params: { path: { uuid: trackUuid } },
      })
    } else {
      await starMutation.mutateAsync({ params: { path: { uuid: trackUuid } } })
    }
    await queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
  }

  // Selection mode state.
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [lastClickedIndex, setLastClickedIndex] = useState<number | null>(null)
  const [bulkSport, setBulkSport] = useState("")
  const [bulkSubSport, setBulkSubSport] = useState("")
  const [bulkTags, setBulkTags] = useState<string[]>([])
  const [toastError, setToastError] = useState<{
    key: number
    msg: string
  } | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const selectionActive = selected.size > 0

  const bulkEditMutation = $api.useMutation("patch", "/tracks")

  function showBulkError(e: unknown) {
    setToastError((prev) => ({
      key: (prev?.key ?? 0) + 1,
      msg: (e as Error).message,
    }))
  }

  const selectedUuids = useMemo(() => [...selected], [selected])

  function clearSelection() {
    setSelected(new Set())
    setLastClickedIndex(null)
    setConfirmDelete(false)
    setBulkSport("")
    setBulkSubSport("")
    setBulkTags([])
  }

  /** Handles clicking a track card in selection mode, with shift-range support. */
  const handleTrackClick = useCallback(
    (
      e: React.MouseEvent,
      uuid: string,
      index: number,
      tracks: { uuid: string }[]
    ) => {
      e.preventDefault()
      e.stopPropagation()
      if (e.shiftKey && lastClickedIndex !== null) {
        const start = Math.min(lastClickedIndex, index)
        const end = Math.max(lastClickedIndex, index)
        setSelected((prev) => {
          const next = new Set(prev)
          for (let i = start; i <= end; i++) {
            next.add(tracks[i].uuid)
          }
          return next
        })
      } else {
        setSelected((prev) => {
          const next = new Set(prev)
          if (next.has(uuid)) {
            next.delete(uuid)
          } else {
            next.add(uuid)
          }
          return next
        })
      }
      setLastClickedIndex(index)
    },
    [lastClickedIndex]
  )

  function selectAll(tracks: { uuid: string }[]) {
    setSelected(new Set(tracks.map((t) => t.uuid)))
  }

  function bulkSetVisibility(isPublic: boolean) {
    if (selectedUuids.length === 0) return
    bulkEditMutation.mutate(
      { body: { uuids: selectedUuids, public: isPublic } },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
        },
        onError: showBulkError,
      }
    )
  }

  function bulkSetTrackType(trackType: number) {
    if (selectedUuids.length === 0) return
    bulkEditMutation.mutate(
      { body: { uuids: selectedUuids, trackType } },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
        },
        onError: showBulkError,
      }
    )
  }

  function bulkApplySport() {
    if (selectedUuids.length === 0 || !bulkSport) return
    const body: Parameters<typeof bulkEditMutation.mutate>[0]["body"] = {
      uuids: selectedUuids,
      sport: parseInt(bulkSport),
    }
    if (bulkSubSport !== "") body.subSport = parseInt(bulkSubSport)
    bulkEditMutation.mutate(
      { body },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
        },
        onError: showBulkError,
      }
    )
  }

  function bulkApplyTags() {
    if (selectedUuids.length === 0 || bulkTags.length === 0) return
    bulkEditMutation.mutate(
      { body: { uuids: selectedUuids, tags: bulkTags } },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
        },
        onError: showBulkError,
      }
    )
  }

  async function bulkDelete() {
    if (selectedUuids.length === 0) return
    try {
      await fetchClient.POST("/tracks/bulk-delete", {
        body: { uuids: selectedUuids },
      })
      clearSelection()
      await queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
      await queryClient.invalidateQueries({
        queryKey: ["get", "/tracks/statistics"],
      })
    } catch (e) {
      showBulkError(e)
    }
  }

  const { data, isLoading, error } = $api.useQuery("get", "/tracks", {
    params: {
      query: {
        page,
        pageSize: pageSize,
        onlyMine,
        ...(publicParam !== undefined ? { public: publicParam } : {}),
        ...(trackTypeParam ? { trackType: trackTypeParam } : {}),
        ...(applied.onlyStarred && user ? { onlyStarred: true } : {}),
        ...(applied.search ? { name: applied.search } : {}),
        ...(applied.sports.length > 0 ? { sport: applied.sports } : {}),
        ...(applied.subSports.length > 0
          ? { subSport: applied.subSports }
          : {}),
        ...(applied.tags.length > 0 ? { tag: applied.tags } : {}),
        ...(applied.tags.length > 1 ? { tagsAnd: applied.tagsAnd } : {}),
        ...(appliedDistMin > 0
          ? { totalDistanceMMin: appliedDistMin * 1000 }
          : {}),
        ...(absMaxDistKm > 0 && appliedDistMax < absMaxDistKm
          ? { totalDistanceMMax: appliedDistMax * 1000 }
          : {}),
        ...(appliedAscentMin > 0 ? { totalAscentMMin: appliedAscentMin } : {}),
        ...(absMaxAscentM > 0 && appliedAscentMax < absMaxAscentM
          ? { totalAscentMMax: appliedAscentMax }
          : {}),
        sortBy: applied.sortBy,
        sortOrder: applied.sortOrder,
      },
    },
  })

  const totalPages = data ? Math.ceil(data.totalCount / pageSize) : 1

  useEffect(() => {
    if (!selectionActive) return
    function onDocumentClick(e: MouseEvent) {
      const target = e.target as HTMLElement
      if (
        target.closest(
          "[data-track-card], [data-bulk-toolbar], button, a, input, select"
        )
      )
        return
      clearSelection()
    }
    document.addEventListener("click", onDocumentClick)
    return () => document.removeEventListener("click", onDocumentClick)
  })

  return (
    <PageContainer className="py-10">
      <h1 className="mb-4 text-xl font-semibold text-text">
        {mode === "public" ? "Public Tracks" : "My Tracks"}
      </h1>

      <div className="mb-6 rounded-lg border border-border bg-panel px-4 pb-4 pt-3">
        <div className="mb-4 flex flex-wrap items-center gap-3">
          <div className="w-56">
            <Input
              type="search"
              placeholder="Search by name or location..."
              value={live.search}
              onChange={(e) =>
                setLive((prev) => ({ ...prev, search: e.target.value }))
              }
              aria-label="Search tracks"
            />
          </div>
          {mode === "user" && (
            <ToggleGroup
              options={[
                { value: "all", label: "All" },
                { value: "public", label: "Public" },
                { value: "private", label: "Private" },
              ]}
              value={live.visibility}
              onChange={(v) => setLive((prev) => ({ ...prev, visibility: v }))}
              ariaLabel="Visibility filter"
            />
          )}
          <ToggleGroup
            options={[
              { value: "all", label: "All" },
              { value: "recorded", label: "Recorded" },
              { value: "planned", label: "Planned" },
            ]}
            value={live.trackType}
            onChange={(v) => setLive((prev) => ({ ...prev, trackType: v }))}
            ariaLabel="Track type filter"
          />
          {user && (
            <button
              onClick={() =>
                setLive((prev) => ({
                  ...prev,
                  onlyStarred: !prev.onlyStarred,
                }))
              }
              aria-pressed={live.onlyStarred}
              className={`cursor-pointer rounded border px-3 py-1.5 text-sm transition-colors ${
                live.onlyStarred
                  ? "border-active bg-active text-active-text"
                  : "border-border text-text-secondary hover:bg-surface"
              }`}
            >
              Starred
            </button>
          )}
          <div className="ml-auto flex items-center gap-2">
            <Select
              value={live.sortBy}
              onChange={(e) =>
                setLive((prev) => ({
                  ...prev,
                  sortBy: e.target.value as SortBy,
                }))
              }
              className="px-2 py-1.5 text-sm"
              aria-label="Sort by"
            >
              {SORT_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </Select>
            <button
              onClick={() =>
                setLive((prev) => ({
                  ...prev,
                  sortOrder: prev.sortOrder === "asc" ? "desc" : "asc",
                }))
              }
              className="cursor-pointer rounded border border-border px-2 py-1.5 text-sm text-text-secondary hover:border-border-hover transition-colors"
              title={live.sortOrder === "asc" ? "Ascending" : "Descending"}
              aria-label={`Sort order: ${live.sortOrder === "asc" ? "ascending" : "descending"}`}
            >
              {live.sortOrder === "asc" ? "A-Z" : "Z-A"}
            </button>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-6">
          {absMaxDistKm > 0 && (
            <div>
              <SectionHeading className="mb-3">Distance</SectionHeading>
              <DualRangeSlider
                absoluteMin={0}
                absoluteMax={absMaxDistKm}
                valueMin={distMin}
                valueMax={distMax}
                step={1}
                formatValue={(v) => `${v} km`}
                onChange={setDistRange}
              />
            </div>
          )}
          {absMaxAscentM > 0 && (
            <div>
              <SectionHeading className="mb-3">Ascent</SectionHeading>
              <DualRangeSlider
                absoluteMin={0}
                absoluteMax={absMaxAscentM}
                valueMin={ascentMin}
                valueMax={ascentMax}
                step={10}
                formatValue={(v) => `${v} m`}
                onChange={setAscentRange}
              />
            </div>
          )}
        </div>

        <div className="mt-4 grid grid-cols-2 gap-6">
          <div>
            <SectionHeading className="mb-2">Sport</SectionHeading>
            <div className="flex flex-wrap gap-1.5">
              {SPORT_IDS.map((id) => (
                <button
                  key={id}
                  onClick={() => toggleSport(id)}
                  aria-pressed={live.sports.includes(id)}
                  className={`cursor-pointer rounded border px-2.5 py-1 text-xs transition-colors ${
                    live.sports.includes(id)
                      ? "border-active bg-active text-active-text"
                      : "border-border text-text-secondary hover:border-border-hover"
                  }`}
                >
                  {SPORT_LABELS[id]}
                </button>
              ))}
            </div>
            {availableSubSports.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1.5">
                {availableSubSports.map((id) => (
                  <button
                    key={id}
                    onClick={() => toggleSubSport(id)}
                    aria-pressed={live.subSports.includes(id)}
                    className={`cursor-pointer rounded border px-2.5 py-1 text-xs transition-colors ${
                      live.subSports.includes(id)
                        ? "border-primary bg-primary/80 text-primary-text"
                        : "border-border text-text-muted hover:border-border-hover"
                    }`}
                  >
                    {SUB_SPORT_LABELS[id]}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div>
            <div className="mb-2 flex items-center justify-between">
              <SectionHeading>Tags</SectionHeading>
              {live.tags.length > 1 && (
                <ToggleGroup
                  options={[
                    { value: "or", label: "OR" },
                    { value: "and", label: "AND" },
                  ]}
                  value={live.tagsAnd ? "and" : "or"}
                  onChange={(v) =>
                    setLive((prev) => ({ ...prev, tagsAnd: v === "and" }))
                  }
                  ariaLabel="Tag match mode"
                  className="text-xs"
                />
              )}
            </div>
            <TagsInput
              value={live.tags}
              onChange={(tags) => setLive((prev) => ({ ...prev, tags }))}
              placeholder="Filter by tag..."
            />
          </div>
        </div>
      </div>

      {isLoading && <p className="text-text-muted">Loading...</p>}

      {error && (
        <p role="alert" className="text-error">
          {(error as unknown as Error).message}
        </p>
      )}

      {data && (
        <>
          {selectionActive && onlyMine && (
            <div
              data-bulk-toolbar
              className="mb-4 rounded-lg border border-primary bg-panel px-4 py-3"
            >
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-medium text-text">
                  {selected.size} selected
                </span>
                <button
                  onClick={() => selectAll(data.tracks)}
                  className="cursor-pointer text-xs text-text-muted hover:text-text-secondary transition-colors"
                >
                  Select all on page
                </button>
                <button
                  onClick={clearSelection}
                  className="cursor-pointer text-xs text-text-muted hover:text-text-secondary transition-colors"
                >
                  Clear
                </button>
                <span className="text-border">|</span>
                <button
                  onClick={() => bulkSetVisibility(true)}
                  disabled={bulkEditMutation.isPending}
                  className="cursor-pointer text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
                >
                  Set public
                </button>
                <button
                  onClick={() => bulkSetVisibility(false)}
                  disabled={bulkEditMutation.isPending}
                  className="cursor-pointer text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
                >
                  Set private
                </button>
                <span className="text-border">|</span>
                <button
                  onClick={() => bulkSetTrackType(2)}
                  disabled={bulkEditMutation.isPending}
                  className="cursor-pointer text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
                >
                  Set recorded
                </button>
                <button
                  onClick={() => bulkSetTrackType(1)}
                  disabled={bulkEditMutation.isPending}
                  className="cursor-pointer text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
                >
                  Set planned
                </button>
                <span className="text-border">|</span>
                <Select
                  value={bulkSport}
                  onChange={(e) => {
                    setBulkSport(e.target.value)
                    setBulkSubSport("")
                  }}
                  className="px-1.5 py-0.5 text-xs"
                >
                  <option value="">Sport...</option>
                  {Object.entries(SPORT_LABELS).map(([id, label]) => (
                    <option key={id} value={id}>
                      {label}
                    </option>
                  ))}
                </Select>
                {bulkSport !== "" && (
                  <Select
                    value={bulkSubSport}
                    onChange={(e) => setBulkSubSport(e.target.value)}
                    className="px-1.5 py-0.5 text-xs"
                  >
                    <option value="">Sub-sport...</option>
                    {(SUB_SPORTS_BY_SPORT[parseInt(bulkSport)] ?? []).map(
                      (id) => (
                        <option key={id} value={String(id)}>
                          {SUB_SPORT_LABELS[id]}
                        </option>
                      )
                    )}
                  </Select>
                )}
                <button
                  onClick={bulkApplySport}
                  disabled={!bulkSport || bulkEditMutation.isPending}
                  className="cursor-pointer text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
                >
                  Set sport
                </button>
                <span className="text-border">|</span>
                <div className="flex min-w-48 flex-1 items-center gap-2">
                  <TagsInput value={bulkTags} onChange={setBulkTags} />
                  <button
                    onClick={bulkApplyTags}
                    disabled={
                      bulkTags.length === 0 || bulkEditMutation.isPending
                    }
                    className="shrink-0 cursor-pointer text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
                  >
                    Set tags
                  </button>
                </div>
              </div>
              <div className="mt-2 flex items-center gap-2">
                {confirmDelete ? (
                  <>
                    <span className="text-xs text-error">
                      Delete {selected.size} track
                      {selected.size > 1 ? "s" : ""}?
                    </span>
                    <Button
                      variant="danger"
                      onClick={() => void bulkDelete()}
                      className="px-2 py-0.5 text-xs"
                    >
                      Confirm delete
                    </Button>
                    <button
                      onClick={() => setConfirmDelete(false)}
                      className="cursor-pointer text-xs text-text-muted hover:text-text-secondary transition-colors"
                    >
                      Cancel
                    </button>
                  </>
                ) : (
                  <Button
                    variant="danger"
                    onClick={() => setConfirmDelete(true)}
                    className="px-2 py-0.5 text-xs"
                  >
                    Delete selected
                  </Button>
                )}
              </div>
            </div>
          )}

          {data.tracks.length === 0 ? (
            <p className="py-12 text-center text-text-muted">
              No tracks found.
            </p>
          ) : (
            <div className="grid grid-cols-2 gap-6 sm:grid-cols-3 lg:grid-cols-4">
              {data.tracks.map((track, index) => {
                const isSelected = selected.has(track.uuid)
                const canSelect = onlyMine && track.isOwner

                const cardContent = (
                  <>
                    <SvgIcon
                      svg={cardCornerSvg}
                      className="tarot-corner -top-0.5 -left-0.5"
                    />
                    <SvgIcon
                      svg={cardCornerSvg}
                      className="tarot-corner -top-0.5 -right-0.5 -scale-x-100"
                    />
                    <SvgIcon
                      svg={cardCornerSvg}
                      className="tarot-corner -bottom-0.5 -left-0.5 -scale-y-100"
                    />
                    <SvgIcon
                      svg={cardCornerSvg}
                      className="tarot-corner -bottom-0.5 -right-0.5 -scale-x-100 -scale-y-100"
                    />

                    <div className="tarot-card-inner">
                      {canSelect && (
                        <button
                          onClick={(e) => {
                            e.preventDefault()
                            e.stopPropagation()
                            handleTrackClick(e, track.uuid, index, data.tracks)
                          }}
                          className={`absolute bottom-2 right-2 z-10 flex h-5 w-5 cursor-pointer items-center justify-center rounded border transition-colors ${
                            isSelected
                              ? "border-primary bg-primary text-primary-text"
                              : "border-border bg-panel/80 text-transparent hover:border-border-hover hover:text-text-muted"
                          }`}
                          aria-label={
                            isSelected ? "Deselect track" : "Select track"
                          }
                          aria-pressed={isSelected}
                        >
                          <svg
                            viewBox="0 0 16 16"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="2"
                            className="h-3 w-3"
                          >
                            <path d="M3 8l3.5 3.5L13 4" />
                          </svg>
                        </button>
                      )}
                      {user && (
                        <button
                          onClick={(e) =>
                            toggleStar(e, track.uuid, track.starred ?? false)
                          }
                          className="absolute right-3 top-3 z-10 cursor-pointer rounded bg-panel/80 p-1 hover:bg-panel transition-colors"
                          aria-label={
                            track.starred ? "Unstar track" : "Star track"
                          }
                        >
                          <StarIcon
                            className={`h-4 w-4 ${track.starred ? "text-star" : "text-text-muted"}`}
                          />
                        </button>
                      )}
                      <div className="aspect-square overflow-hidden bg-surface text-track">
                        <SvgPreview
                          src={`/api/tracks/${track.uuid}/preview.svg?size=256`}
                          alt="Track preview"
                          className="h-full w-full object-contain"
                        />
                      </div>
                      <div className="px-2.5 pb-2.5">
                        <div className="flex items-center gap-1.5">
                          <img
                            src={`/api/users/${track.user.uuid}/avatar`}
                            alt=""
                            className="h-4 w-4 shrink-0 rounded-full"
                          />
                          <p className="truncate font-[Fondamento] text-sm font-medium text-text">
                            {track.name}
                          </p>
                        </div>
                        <div className="mt-1 flex items-center gap-2 text-xs text-text-muted">
                          <span>{track.user.name}</span>
                          <span className="flex items-center gap-0.5">
                            <SvgIcon
                              svg={distanceSvg}
                              className="inline h-3 w-3"
                            />
                            {formatDistance(track.totalDistanceM)}
                          </span>
                          <span className="flex items-center gap-0.5">
                            <SvgIcon
                              svg={elevationSvg}
                              className="inline h-3 w-3"
                            />
                            {formatAscent(track.totalAscentM)}
                          </span>
                        </div>
                        <div className="mt-1.5 overflow-hidden rounded bg-surface text-track">
                          <SvgPreview
                            src={`/api/tracks/${track.uuid}/profile.svg?size=256`}
                            alt="Elevation profile"
                            className="w-full"
                          />
                        </div>
                        {track.forecast && (
                          <div
                            className="mt-1.5 flex items-center gap-x-2 text-xs"
                            title={`Forecast: ${new Date(track.forecast.forecastReferenceTime).toLocaleString()}\nStart: ${new Date(track.forecast.startTime).toLocaleString()}`}
                          >
                            {track.forecast.avgTemperatureC != null && (
                              <span className="flex items-center gap-0.5 text-error">
                                <SvgIcon
                                  svg={temperatureSvg}
                                  className="inline h-3 w-3"
                                />
                                {track.forecast.avgTemperatureC.toFixed(0)}
                                &deg;C
                              </span>
                            )}
                            {track.forecast.totalPrecipitationMm != null && (
                              <span className="flex items-center gap-0.5 text-info">
                                <SvgIcon
                                  svg={rainSvg}
                                  className="inline h-3 w-3"
                                />
                                {track.forecast.totalPrecipitationMm < 0.1
                                  ? "dry"
                                  : `${track.forecast.totalPrecipitationMm.toFixed(1)} mm`}
                              </span>
                            )}
                            <MiniWindRose
                              head={track.forecast.windHeadMs}
                              right={track.forecast.windRightMs}
                              tail={track.forecast.windTailMs}
                              left={track.forecast.windLeftMs}
                            />
                          </div>
                        )}
                      </div>
                    </div>
                  </>
                )

                if (selectionActive && canSelect) {
                  return (
                    <div
                      data-track-card
                      key={track.uuid}
                      onClick={(e) =>
                        handleTrackClick(e, track.uuid, index, data.tracks)
                      }
                      className={`tarot-card group relative block cursor-pointer ${
                        isSelected
                          ? "ring-2 ring-primary ring-offset-2 ring-offset-surface"
                          : ""
                      }`}
                      role="checkbox"
                      aria-checked={isSelected}
                      aria-label={`Select ${track.name}`}
                    >
                      {cardContent}
                    </div>
                  )
                }

                return (
                  <Link
                    data-track-card
                    key={track.uuid}
                    to={`/tracks/${track.uuid}`}
                    className={`tarot-card group relative block ${
                      isSelected
                        ? "ring-2 ring-primary ring-offset-2 ring-offset-surface"
                        : ""
                    }`}
                  >
                    {cardContent}
                  </Link>
                )
              })}
            </div>
          )}

          {(totalPages > 1 || pageSize !== DEFAULT_PAGE_SIZE) && (
            <div className="mt-8 flex items-center justify-center gap-4">
              <Button
                variant="secondary"
                disabled={page === 1}
                onClick={() => setPage((p) => p - 1)}
                className="px-3 py-1.5"
              >
                Previous
              </Button>
              <span className="text-sm text-text-secondary">
                {page} / {totalPages}
              </span>
              <Button
                variant="secondary"
                disabled={page === totalPages}
                onClick={() => setPage((p) => p + 1)}
                className="px-3 py-1.5"
              >
                Next
              </Button>
              <Select
                value={String(pageSize)}
                onChange={(e) => {
                  setPageSize(Number(e.target.value))
                }}
                className="px-2 py-1.5 text-sm"
                aria-label="Page size"
              >
                {PAGE_SIZE_OPTIONS.map((size) => (
                  <option key={size} value={size}>
                    {size} / page
                  </option>
                ))}
              </Select>
            </div>
          )}
        </>
      )}
      {toastError && (
        <Toast
          key={toastError.key}
          message={toastError.msg}
          onDismiss={() => setToastError(null)}
        />
      )}
    </PageContainer>
  )
}
