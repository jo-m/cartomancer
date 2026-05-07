import { $api } from "../api/client"
import { useSession } from "../context/SessionContext"
import {
  SPORT_LABELS,
  SUB_SPORT_LABELS,
  SUB_SPORTS_BY_SPORT,
} from "../lib/sports"
import TagsInput from "./TagsInput"
import DualRangeSlider from "./ui/DualRangeSlider"
import Input from "./ui/Input"
import Select from "./ui/Select"
import ToggleGroup from "./ui/ToggleGroup"
import SectionHeading from "./ui/SectionHeading"
import type { SortBy } from "./TrackGrid"

const SORT_OPTIONS: { value: SortBy; label: string }[] = [
  { value: "created_at", label: "Uploaded at" },
  { value: "total_distance_m", label: "Distance" },
  { value: "total_ascent_m", label: "Ascent" },
]

const SPORT_IDS = [1, 2, 0] as const

export interface LiveFilters {
  search: string
  distRange: [number, number] | null
  ascentRange: [number, number] | null
  visibility: "all" | "public" | "private"
  trackType: "all" | "recorded" | "planned"
  onlyStarred: boolean
  sports: number[]
  subSports: number[]
  tags: string[]
  tagsAnd: boolean
  sortBy: SortBy
  sortOrder: "asc" | "desc"
  /**
   * Optional radial start-location filter. When set, only tracks whose
   * recorded start point lies within `radiusM` of (lat, lon) are returned.
   * The picker UI lives in the map view; this field is shared so the same
   * URL/query state is used by both views.
   */
  startNear: { lat: number; lon: number; radiusM: number } | null
}

export interface TrackFiltersProps {
  mode: "public" | "user"
  live: LiveFilters
  setLive: React.Dispatch<React.SetStateAction<LiveFilters>>
  absMaxDistKm: number
  absMaxAscentM: number
}

/** Filter panel for the track grid with search, toggles, sliders, sport/tag filters, and sorting. */
export default function TrackFilters({
  mode,
  live,
  setLive,
  absMaxDistKm,
  absMaxAscentM,
}: TrackFiltersProps) {
  const { user } = useSession()

  const distMin = live.distRange?.[0] ?? 0
  const distMax = live.distRange?.[1] ?? absMaxDistKm
  const ascentMin = live.ascentRange?.[0] ?? 0
  const ascentMax = live.ascentRange?.[1] ?? absMaxAscentM

  function setDistRange(min: number, max: number) {
    const r: [number, number] | null =
      min === 0 && max === absMaxDistKm ? null : [min, max]
    setLive((prev) => ({ ...prev, distRange: r }))
  }

  function setAscentRange(min: number, max: number) {
    const r: [number, number] | null =
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
      : [...new Set(live.sports.flatMap((s) => SUB_SPORTS_BY_SPORT[s] ?? []))]

  const tagSource = mode === "public" ? "public" : "user"
  const userTagsQuery = $api.useQuery(
    "get",
    "/tags",
    {},
    { enabled: tagSource === "user" && !!user }
  )
  const publicTagsQuery = $api.useQuery(
    "get",
    "/tags/public",
    {},
    { enabled: tagSource === "public" }
  )
  const allTagsData =
    tagSource === "public" ? publicTagsQuery.data : userTagsQuery.data
  const availableTags = (allTagsData?.tags ?? []).filter(
    (t) => !live.tags.includes(t.tag)
  )

  function addTag(tag: string) {
    setLive((prev) =>
      prev.tags.includes(tag) ? prev : { ...prev, tags: [...prev.tags, tag] }
    )
  }

  return (
    <div className="mb-6 rounded-lg border border-border bg-panel px-4 pb-4 pt-3">
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="w-56">
          <Input
            type="search"
            placeholder="Search by name, location or filename..."
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
            source={tagSource}
          />
          {availableTags.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-1">
              {availableTags.map(({ tag, nTracks }) => (
                <button
                  key={tag}
                  type="button"
                  onClick={() => addTag(tag)}
                  className="cursor-pointer inline-flex items-center gap-1.5 rounded border border-border py-0.5 pl-2 pr-1 text-xs text-text-secondary hover:border-border-hover hover:bg-surface transition-colors"
                  aria-label={`Add tag ${tag} to filter`}
                >
                  <span className="font-medium text-text">{tag}</span>
                  <span className="rounded bg-surface px-1 text-[10px] tabular-nums text-text-muted">
                    {nTracks}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
