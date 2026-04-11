import { useState, useEffect, useRef, useMemo, useCallback } from "react"
import { $api } from "../api/client"
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
import { useSession } from "../context/SessionContext"
import Toast from "./Toast"
import useToast from "../hooks/useToast"
import Button from "./ui/Button"
import Select from "./ui/Select"
import PageContainer from "./ui/PageContainer"
import { useQueryClient } from "@tanstack/react-query"
import TrackFilters from "./TrackFilters"
import type { LiveFilters } from "./TrackFilters"
import TrackCard from "./TrackCard"
import BulkEditToolbar from "./BulkEditToolbar"

const DEFAULT_PAGE_SIZE = 24
const PAGE_SIZE_OPTIONS = [12, 24, 48, 96]

export type SortBy = "created_at" | "total_distance_m" | "total_ascent_m"
export type SortOrder = "asc" | "desc"

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
  const {
    toast: toastError,
    showToast: showBulkToast,
    dismissToast: dismissBulkError,
  } = useToast()

  const selectionActive = selected.size > 0

  function showBulkError(e: unknown) {
    showBulkToast((e as Error).message)
  }

  function clearSelection() {
    setSelected(new Set())
    setLastClickedIndex(null)
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

  /** Handles clicking a track card in selection mode, with shift-range support. */
  const handleTrackClick = useCallback(
    (e: React.MouseEvent, uuid: string, index: number) => {
      e.preventDefault()
      e.stopPropagation()
      if (e.shiftKey && lastClickedIndex !== null && data) {
        const start = Math.min(lastClickedIndex, index)
        const end = Math.max(lastClickedIndex, index)
        setSelected((prev) => {
          const next = new Set(prev)
          for (let i = start; i <= end; i++) {
            next.add(data.tracks[i].uuid)
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
    [lastClickedIndex, data]
  )

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
  }, [selectionActive])

  return (
    <PageContainer>
      <h1 className="mb-4 text-xl font-semibold text-text">
        {mode === "public" ? "Public Tracks" : "My Tracks"}
      </h1>

      <TrackFilters
        mode={mode}
        live={live}
        setLive={setLive}
        absMaxDistKm={absMaxDistKm}
        absMaxAscentM={absMaxAscentM}
      />

      {isLoading && <p className="text-text-muted">Loading...</p>}

      {error && (
        <p role="alert" className="text-error">
          {error.message}
        </p>
      )}

      {data && (
        <>
          {selectionActive && onlyMine && (
            <BulkEditToolbar
              selected={selected}
              onSelectAll={() =>
                setSelected(new Set(data.tracks.map((t) => t.uuid)))
              }
              onClearSelection={clearSelection}
              onError={showBulkError}
            />
          )}

          {data.tracks.length === 0 ? (
            <p className="py-12 text-center text-text-muted">
              No tracks found.
            </p>
          ) : (
            <div className="grid grid-cols-2 gap-6 sm:grid-cols-3 lg:grid-cols-4">
              {data.tracks.map((track, index) => (
                <TrackCard
                  key={track.uuid}
                  track={track}
                  index={index}
                  isSelected={selected.has(track.uuid)}
                  selectionActive={selectionActive}
                  canSelect={onlyMine && (track.isOwner ?? false)}
                  showStar={!!user}
                  onToggleStar={toggleStar}
                  onSelect={handleTrackClick}
                />
              ))}
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
          message={toastError.message}
          variant={toastError.variant}
          onDismiss={dismissBulkError}
        />
      )}
    </PageContainer>
  )
}
