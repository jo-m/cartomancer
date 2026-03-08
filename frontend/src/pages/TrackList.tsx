import { useState, useEffect } from "react"
import { Link } from "react-router-dom"
import { $api } from "../api/client"

const PAGE_SIZE = 24

function formatDistance(m: number): string {
  return `${(m / 1000).toFixed(1)} km`
}

function formatAscent(m: number): string {
  return `${Math.round(m)} m`
}

interface Filters {
  search: string
  distMinEnabled: boolean
  distMaxEnabled: boolean
  distMinKm: string
  distMaxKm: string
  ascentMinEnabled: boolean
  ascentMaxEnabled: boolean
  ascentMinM: string
  ascentMaxM: string
}

const emptyFilters: Filters = {
  search: "",
  distMinEnabled: false,
  distMaxEnabled: false,
  distMinKm: "",
  distMaxKm: "",
  ascentMinEnabled: false,
  ascentMaxEnabled: false,
  ascentMinM: "",
  ascentMaxM: "",
}

function buildQueryParams(f: Filters) {
  const distMinM =
    f.distMinEnabled && f.distMinKm !== ""
      ? parseFloat(f.distMinKm) * 1000
      : undefined
  const distMaxM =
    f.distMaxEnabled && f.distMaxKm !== ""
      ? parseFloat(f.distMaxKm) * 1000
      : undefined
  const ascentMinM =
    f.ascentMinEnabled && f.ascentMinM !== ""
      ? parseFloat(f.ascentMinM)
      : undefined
  const ascentMaxM =
    f.ascentMaxEnabled && f.ascentMaxM !== ""
      ? parseFloat(f.ascentMaxM)
      : undefined

  return {
    ...(f.search ? { name: f.search } : {}),
    ...(distMinM !== undefined ? { totalDistanceMMin: distMinM } : {}),
    ...(distMaxM !== undefined ? { totalDistanceMMax: distMaxM } : {}),
    ...(ascentMinM !== undefined ? { totalAscentMMin: ascentMinM } : {}),
    ...(ascentMaxM !== undefined ? { totalAscentMMax: ascentMaxM } : {}),
  }
}

interface RangeFilterProps {
  label: string
  unit: string
  minEnabled: boolean
  maxEnabled: boolean
  minValue: string
  maxValue: string
  rangeHint?: string
  onMinEnabledChange: (v: boolean) => void
  onMaxEnabledChange: (v: boolean) => void
  onMinValueChange: (v: string) => void
  onMaxValueChange: (v: string) => void
}

function RangeFilter({
  label,
  unit,
  minEnabled,
  maxEnabled,
  minValue,
  maxValue,
  rangeHint,
  onMinEnabledChange,
  onMaxEnabledChange,
  onMinValueChange,
  onMaxValueChange,
}: RangeFilterProps) {
  return (
    <div className="flex flex-wrap items-center gap-3 text-sm">
      <span className="w-16 shrink-0 font-medium text-gray-700">{label}</span>
      <label className="flex items-center gap-1.5">
        <input
          type="checkbox"
          checked={minEnabled}
          onChange={(e) => onMinEnabledChange(e.target.checked)}
          className="accent-gray-700"
        />
        <span className="text-gray-600">Min</span>
        <input
          type="number"
          min={0}
          value={minValue}
          disabled={!minEnabled}
          onChange={(e) => onMinValueChange(e.target.value)}
          className="w-20 rounded border border-gray-300 px-2 py-1 text-sm disabled:opacity-40"
          placeholder="0"
        />
        <span className="text-gray-500">{unit}</span>
      </label>
      <label className="flex items-center gap-1.5">
        <input
          type="checkbox"
          checked={maxEnabled}
          onChange={(e) => onMaxEnabledChange(e.target.checked)}
          className="accent-gray-700"
        />
        <span className="text-gray-600">Max</span>
        <input
          type="number"
          min={0}
          value={maxValue}
          disabled={!maxEnabled}
          onChange={(e) => onMaxValueChange(e.target.value)}
          className="w-20 rounded border border-gray-300 px-2 py-1 text-sm disabled:opacity-40"
          placeholder="∞"
        />
        <span className="text-gray-500">{unit}</span>
      </label>
      {rangeHint && <span className="text-xs text-gray-400">{rangeHint}</span>}
    </div>
  )
}

export default function TrackList() {
  const [liveFilters, setLiveFilters] = useState<Filters>(emptyFilters)
  const [appliedFilters, setAppliedFilters] = useState<Filters>(emptyFilters)
  const [page, setPage] = useState(1)

  useEffect(() => {
    const timer = setTimeout(() => {
      setAppliedFilters(liveFilters)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [liveFilters])

  const { data: stats } = $api.useQuery("get", "/tracks/statistics")

  const { data, isLoading, error } = $api.useQuery("get", "/tracks", {
    params: {
      query: {
        page,
        pageSize: PAGE_SIZE,
        ...buildQueryParams(appliedFilters),
      },
    },
  })

  function set<K extends keyof Filters>(key: K, value: Filters[K]) {
    setLiveFilters((prev) => ({ ...prev, [key]: value }))
  }

  const distRangeHint =
    stats?.totalDistanceMMin != null && stats?.totalDistanceMMax != null
      ? `Range: ${formatDistance(stats.totalDistanceMMin)} – ${formatDistance(stats.totalDistanceMMax)}`
      : undefined

  const ascentRangeHint =
    stats?.totalAscentMMin != null && stats?.totalAscentMMax != null
      ? `Range: ${formatAscent(stats.totalAscentMMin)} – ${formatAscent(stats.totalAscentMMax)}`
      : undefined

  const totalPages = data ? Math.ceil(data.totalCount / PAGE_SIZE) : 1

  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <div className="mb-4 flex items-center justify-between gap-4">
        <h1 className="text-xl font-semibold text-gray-900">Tracks</h1>
        <input
          type="search"
          placeholder="Search by name…"
          value={liveFilters.search}
          onChange={(e) => set("search", e.target.value)}
          className="w-64 rounded border border-gray-300 px-3 py-1.5 text-sm focus:border-gray-500 focus:outline-none"
        />
      </div>

      <div className="mb-6 space-y-2 rounded-lg border border-gray-200 bg-white p-3">
        <RangeFilter
          label="Distance"
          unit="km"
          minEnabled={liveFilters.distMinEnabled}
          maxEnabled={liveFilters.distMaxEnabled}
          minValue={liveFilters.distMinKm}
          maxValue={liveFilters.distMaxKm}
          rangeHint={distRangeHint}
          onMinEnabledChange={(v) => set("distMinEnabled", v)}
          onMaxEnabledChange={(v) => set("distMaxEnabled", v)}
          onMinValueChange={(v) => set("distMinKm", v)}
          onMaxValueChange={(v) => set("distMaxKm", v)}
        />
        <RangeFilter
          label="Ascent"
          unit="m"
          minEnabled={liveFilters.ascentMinEnabled}
          maxEnabled={liveFilters.ascentMaxEnabled}
          minValue={liveFilters.ascentMinM}
          maxValue={liveFilters.ascentMaxM}
          rangeHint={ascentRangeHint}
          onMinEnabledChange={(v) => set("ascentMinEnabled", v)}
          onMaxEnabledChange={(v) => set("ascentMaxEnabled", v)}
          onMinValueChange={(v) => set("ascentMinM", v)}
          onMaxValueChange={(v) => set("ascentMaxM", v)}
        />
      </div>

      {isLoading && <p className="text-gray-500">Loading…</p>}

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
                      src={`/api/tracks/${track.uuid}/preview.svg`}
                      alt=""
                      className="h-full w-full object-contain"
                    />
                  </div>
                  <div className="p-2.5">
                    <p className="truncate text-sm font-medium text-gray-900">
                      {track.name}
                    </p>
                    <p className="mt-0.5 text-xs text-gray-500">
                      {formatDistance(track.totalDistanceM)} ·{" "}
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
