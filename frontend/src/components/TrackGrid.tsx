import { useState, useEffect, useRef } from "react"
import { Link } from "react-router-dom"
import { $api } from "../api/client"

const PAGE_SIZE = 24

function formatDistance(m: number): string {
  return `${(m / 1000).toFixed(1)} km`
}

function formatAscent(m: number): string {
  return `${Math.round(m)} m`
}

// DualRangeSlider uses pointer events on the outer container; thumbs are visual only.
// Which thumb to move is decided by proximity to the pointer position.

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

  // The outer div has 8px horizontal padding so thumbs at 0%/100% stay fully visible.
  // Value calculations subtract that padding from the bounding rect.
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

  // Thumb position: calc(frac*100% + (8 - frac*16)px) places the center
  // exactly at the correct inner position while the outer 8px padding absorbs overflow.
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
      >
        <div className="absolute inset-x-2 top-1/2 h-1.5 -translate-y-1/2 rounded-full bg-gray-200" />
        <div
          className="absolute top-1/2 h-1.5 -translate-y-1/2 rounded-full bg-gray-500"
          style={highlightStyle}
        />
        <div
          className="pointer-events-none absolute top-1/2 h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 bg-white"
          style={{
            left: thumbLeft(valueMin),
            borderColor: minActive ? "#6b7280" : "#d1d5db",
          }}
        />
        <div
          className="pointer-events-none absolute top-1/2 h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 bg-white"
          style={{
            left: thumbLeft(valueMax),
            borderColor: maxActive ? "#6b7280" : "#d1d5db",
          }}
        />
      </div>
      <div className="mt-1 flex justify-between text-xs">
        <span
          className={minActive ? "font-medium text-gray-700" : "text-gray-400"}
        >
          {formatValue(valueMin)}
        </span>
        <span
          className={maxActive ? "font-medium text-gray-700" : "text-gray-400"}
        >
          {formatValue(valueMax)}
        </span>
      </div>
    </div>
  )
}

// Slider range state is null when at the full range (= no filter).
type Range = [number, number]

interface LiveFilters {
  search: string
  distRange: Range | null // km; null = full range
  ascentRange: Range | null // m; null = full range
  visibility: "all" | "public" | "private" // user mode only
}

const initialFilters: LiveFilters = {
  search: "",
  distRange: null,
  ascentRange: null,
  visibility: "all",
}

export interface TrackGridProps {
  // mode "public" shows all public tracks; "user" shows only the current user's tracks.
  mode: "public" | "user"
}

/** TrackGrid renders a filterable, paginated grid of track cards. */
export default function TrackGrid({ mode }: TrackGridProps) {
  const onlyMine = mode === "user"

  const { data: stats } = $api.useQuery("get", "/tracks/statistics", {
    params: { query: { onlyMine } },
  })

  const absMaxDistKm =
    stats?.totalDistanceMMax != null
      ? Math.ceil(stats.totalDistanceMMax / 1000)
      : 0
  const absMaxAscentM =
    stats?.totalAscentMMax != null
      ? Math.ceil(stats.totalAscentMMax / 10) * 10
      : 0

  const [live, setLive] = useState<LiveFilters>(initialFilters)
  const [applied, setApplied] = useState<LiveFilters>(initialFilters)
  const [page, setPage] = useState(1)

  useEffect(() => {
    const timer = setTimeout(() => {
      setApplied(live)
      setPage(1)
    }, 200)
    return () => clearTimeout(timer)
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

  const appliedDistMin = applied.distRange?.[0] ?? 0
  const appliedDistMax = applied.distRange?.[1] ?? absMaxDistKm
  const appliedAscentMin = applied.ascentRange?.[0] ?? 0
  const appliedAscentMax = applied.ascentRange?.[1] ?? absMaxAscentM

  // Derive the public query param from mode and user visibility filter.
  let publicParam: boolean | undefined
  if (mode === "public") {
    publicParam = true
  } else if (applied.visibility === "public") {
    publicParam = true
  } else if (applied.visibility === "private") {
    publicParam = false
  }

  const { data, isLoading, error } = $api.useQuery("get", "/tracks", {
    params: {
      query: {
        page,
        pageSize: PAGE_SIZE,
        onlyMine,
        ...(publicParam !== undefined ? { public: publicParam } : {}),
        ...(applied.search ? { name: applied.search } : {}),
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
      },
    },
  })

  const totalPages = data ? Math.ceil(data.totalCount / PAGE_SIZE) : 1

  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <div className="mb-4 flex items-center justify-between gap-4">
        <h1 className="text-xl font-semibold text-gray-900">
          {mode === "public" ? "Tracks" : "My Tracks"}
        </h1>
        <div className="flex items-center gap-3">
          {mode === "user" && (
            <div className="flex rounded border border-gray-300 text-sm">
              {(["all", "public", "private"] as const).map((v) => (
                <button
                  key={v}
                  onClick={() =>
                    setLive((prev) => ({ ...prev, visibility: v }))
                  }
                  className={`px-3 py-1.5 first:rounded-l last:rounded-r ${
                    live.visibility === v
                      ? "bg-gray-800 text-white"
                      : "text-gray-700 hover:bg-gray-50"
                  }`}
                >
                  {v === "all" ? "All" : v === "public" ? "Public" : "Private"}
                </button>
              ))}
            </div>
          )}
          <input
            type="search"
            placeholder="Search by name..."
            value={live.search}
            onChange={(e) =>
              setLive((prev) => ({ ...prev, search: e.target.value }))
            }
            className="w-56 rounded border border-gray-300 px-3 py-1.5 text-sm focus:border-gray-500 focus:outline-none"
          />
        </div>
      </div>

      {absMaxDistKm > 0 && (
        <div className="mb-6 grid grid-cols-2 gap-6 rounded-lg border border-gray-200 bg-white px-4 pb-3 pt-3">
          <div>
            <p className="mb-3 text-xs font-medium uppercase tracking-wide text-gray-500">
              Distance
            </p>
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
          {absMaxAscentM > 0 && (
            <div>
              <p className="mb-3 text-xs font-medium uppercase tracking-wide text-gray-500">
                Ascent
              </p>
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
      )}

      {isLoading && <p className="text-gray-500">Loading...</p>}

      {error && (
        <p className="text-red-600">{(error as unknown as Error).message}</p>
      )}

      {data && (
        <>
          {data.tracks.length === 0 ? (
            <p className="py-12 text-center text-gray-500">No tracks found.</p>
          ) : (
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
              {data.tracks.map((track) => (
                <Link
                  key={track.uuid}
                  to={`/tracks/${track.uuid}`}
                  className="group block rounded-lg border border-gray-200 bg-white hover:border-gray-400"
                >
                  <div className="aspect-square overflow-hidden rounded-t-lg bg-gray-50">
                    <img
                      src={`/api/tracks/${track.uuid}/preview.svg?size=256`}
                      alt="Track preview"
                      className="h-full w-full object-contain"
                    />
                  </div>
                  <div className="p-2.5">
                    <p className="truncate text-sm font-medium text-gray-900">
                      {track.name}
                    </p>
                    <p className="mt-0.5 text-xs text-gray-500">
                      {formatDistance(track.totalDistanceM)} &middot;{" "}
                      {formatAscent(track.totalAscentM)}
                    </p>
                  </div>
                </Link>
              ))}
            </div>
          )}

          {totalPages > 1 && (
            <div className="mt-8 flex items-center justify-center gap-4">
              <button
                disabled={page === 1}
                onClick={() => setPage((p) => p - 1)}
                className="rounded border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:border-gray-400 disabled:cursor-not-allowed disabled:opacity-40"
              >
                Previous
              </button>
              <span className="text-sm text-gray-600">
                {page} / {totalPages}
              </span>
              <button
                disabled={page === totalPages}
                onClick={() => setPage((p) => p + 1)}
                className="rounded border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:border-gray-400 disabled:cursor-not-allowed disabled:opacity-40"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
