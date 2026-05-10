import { useState } from "react"
import { useNavigate, Link } from "react-router-dom"
import { useQueryClient } from "@tanstack/react-query"
import { $api } from "../api/client"
import { externalUrl } from "../lib/externalUrl"
import {
  SPORT_LABELS,
  SUB_SPORT_LABELS,
  SUB_SPORTS_BY_SPORT,
} from "../lib/sports"
import SvgIcon from "../assets/SvgIcon"
import distanceSvg from "../assets/distance.svg?raw"
import elevationSvg from "../assets/elevation.svg?raw"
import SectionHeading from "./ui/SectionHeading"
import Badge from "./ui/Badge"
import Button from "./ui/Button"
import ToggleGroup from "./ui/ToggleGroup"
import TagsInput from "./TagsInput"
import { formatDistance, formatAscent } from "../lib/format"
import TimeAgo from "./TimeAgo"

const TRACK_TYPE_LABELS: Record<number, string> = {
  0: "Unknown",
  1: "Planned",
  2: "Recorded",
}

const FILE_FORMAT_LABELS: Record<number, string> = {
  0: "GPX",
  1: "FIT",
}

interface SimilarTrack {
  uuid: string
  name: string
  totalDistanceM: number
}

interface TrackDetailData {
  uuid: string
  name: string
  public?: boolean
  isOwner?: boolean
  totalDistanceM: number
  totalAscentM: number
  sport: number
  subSport: number
  trackType: number
  fileFormat: number
  originalFilename?: string
  originalCreatedAt?: string
  createdAt: string
  author?: string
  authorLinkUrl?: string
  linkUrl?: string
  tags: string[]
  similarTracks: SimilarTrack[]
}

export interface TrackDetailsProps {
  track: TrackDetailData
  onError?: (msg: string) => void
  onSuccess?: (msg: string) => void
}

type EditField = "sport" | "subSport" | "trackType" | "tags" | null

/** Displays track metadata with inline editing for track owners. */
export default function TrackDetails({
  track,
  onError,
  onSuccess,
}: TrackDetailsProps) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const isOwner = track.isOwner ?? false
  const [editingField, setEditingField] = useState<EditField>(null)
  const [editTags, setEditTags] = useState<string[]>([])
  const [confirmDelete, setConfirmDelete] = useState(false)

  const editMutation = $api.useMutation("patch", "/tracks/{uuid}")
  const deleteMutation = $api.useMutation("delete", "/tracks/{uuid}")

  async function saveField(overrides: Record<string, unknown>) {
    try {
      await editMutation.mutateAsync({
        params: { path: { uuid: track.uuid } },
        body: {
          name: track.name,
          public: track.public ?? false,
          trackType: track.trackType,
          sport: track.sport,
          subSport: track.subSport,
          tags: track.tags,
          ...overrides,
        },
      })
      await queryClient.invalidateQueries({
        queryKey: ["get", "/tracks/{uuid}"],
      })
      setEditingField(null)
      onSuccess?.("Track saved.")
    } catch (err) {
      onError?.((err as Error).message)
    }
  }

  function handleSelectChange(field: string, value: number) {
    const overrides: Record<string, unknown> = { [field]: value }
    if (field === "sport") {
      const available = SUB_SPORTS_BY_SPORT[value] ?? [0]
      if (!available.includes(track.subSport)) {
        overrides.subSport = available[0]
      }
    }
    saveField(overrides)
  }

  async function handleDelete() {
    try {
      await deleteMutation.mutateAsync({
        params: { path: { uuid: track.uuid } },
      })
      navigate("/")
    } catch (err) {
      onError?.((err as Error).message)
      setConfirmDelete(false)
    }
  }

  const editableClass = isOwner ? "editable-field" : ""

  const selectClass =
    "min-h-11 rounded border border-border bg-panel px-2 py-1 text-sm text-text focus:border-primary focus:outline-none cursor-pointer transition-colors"

  const subSportOptions = SUB_SPORTS_BY_SPORT[track.sport] ?? [0]
  const showSubSport =
    track.subSport !== 0 || (isOwner && subSportOptions.length > 1)

  return (
    <>
      <dl className="mt-4 grid grid-cols-2 gap-x-6 gap-y-2 sm:grid-cols-4 lg:grid-cols-6 xl:grid-cols-8">
        <div>
          <dt className="flex items-center gap-1 text-xs font-medium uppercase tracking-wide text-text-muted">
            <SvgIcon svg={distanceSvg} className="h-3.5 w-3.5" />
            Distance
          </dt>
          <dd className="mt-0.5 text-sm text-text">
            {formatDistance(track.totalDistanceM)}
          </dd>
        </div>

        <div>
          <dt className="flex items-center gap-1 text-xs font-medium uppercase tracking-wide text-text-muted">
            <SvgIcon svg={elevationSvg} className="h-3.5 w-3.5" />
            Ascent
          </dt>
          <dd className="mt-0.5 text-sm text-text">
            {formatAscent(track.totalAscentM)}
          </dd>
        </div>

        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Sport
          </dt>
          <dd className="mt-0.5 text-sm text-text">
            {editingField === "sport" ? (
              <select
                autoFocus
                value={track.sport}
                onChange={(e) =>
                  handleSelectChange("sport", Number(e.target.value))
                }
                onBlur={() => setEditingField(null)}
                className={selectClass}
                aria-label="Sport"
              >
                {Object.entries(SPORT_LABELS).map(([k, v]) => (
                  <option key={k} value={k}>
                    {v}
                  </option>
                ))}
              </select>
            ) : editingField === "subSport" ? (
              <div className="flex flex-wrap items-center gap-1">
                <span>{SPORT_LABELS[track.sport] ?? track.sport}</span>
                <select
                  autoFocus
                  value={track.subSport}
                  onChange={(e) =>
                    handleSelectChange("subSport", Number(e.target.value))
                  }
                  onBlur={() => setEditingField(null)}
                  className={selectClass}
                  aria-label="Sub-sport"
                >
                  {subSportOptions.map((id) => (
                    <option key={id} value={id}>
                      {SUB_SPORT_LABELS[id]}
                    </option>
                  ))}
                </select>
              </div>
            ) : (
              <span className="inline-flex flex-wrap items-center gap-x-1">
                <span
                  className={editableClass}
                  onClick={() => isOwner && setEditingField("sport")}
                  role={isOwner ? "button" : undefined}
                  tabIndex={isOwner ? 0 : undefined}
                  onKeyDown={(e) => {
                    if (isOwner && e.key === "Enter") setEditingField("sport")
                  }}
                  aria-label={isOwner ? "Edit sport" : undefined}
                >
                  {SPORT_LABELS[track.sport] ?? track.sport}
                </span>
                {showSubSport && (
                  <span
                    className={`text-text-muted ${editableClass}`}
                    onClick={() => isOwner && setEditingField("subSport")}
                    role={isOwner ? "button" : undefined}
                    tabIndex={isOwner ? 0 : undefined}
                    onKeyDown={(e) => {
                      if (isOwner && e.key === "Enter")
                        setEditingField("subSport")
                    }}
                    aria-label={isOwner ? "Edit sub-sport" : undefined}
                  >
                    ({SUB_SPORT_LABELS[track.subSport] ?? track.subSport})
                  </span>
                )}
              </span>
            )}
          </dd>
        </div>

        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Type
          </dt>
          <dd className="mt-0.5 text-sm text-text">
            {editingField === "trackType" ? (
              <select
                autoFocus
                value={track.trackType}
                onChange={(e) =>
                  handleSelectChange("trackType", Number(e.target.value))
                }
                onBlur={() => setEditingField(null)}
                className={selectClass}
                aria-label="Track type"
              >
                <option value={2}>Recorded</option>
                <option value={1}>Planned</option>
              </select>
            ) : (
              <span
                className={editableClass}
                onClick={() => isOwner && setEditingField("trackType")}
                role={isOwner ? "button" : undefined}
                tabIndex={isOwner ? 0 : undefined}
                onKeyDown={(e) => {
                  if (isOwner && e.key === "Enter") setEditingField("trackType")
                }}
                aria-label={isOwner ? "Edit track type" : undefined}
              >
                {TRACK_TYPE_LABELS[track.trackType] ?? track.trackType}
              </span>
            )}
          </dd>
        </div>

        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Format
          </dt>
          <dd className="mt-0.5 text-sm text-text">
            {FILE_FORMAT_LABELS[track.fileFormat] ?? track.fileFormat}
          </dd>
        </div>

        {track.originalFilename && (
          <div className="col-span-2">
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Original filename
            </dt>
            <dd
              className="mt-0.5 truncate text-sm text-text"
              title={track.originalFilename}
            >
              {track.originalFilename}
            </dd>
          </div>
        )}

        {track.originalCreatedAt && (
          <div>
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Activity date
            </dt>
            <dd className="mt-0.5 text-sm text-text">
              <TimeAgo iso={track.originalCreatedAt} />
            </dd>
          </div>
        )}

        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Uploaded
          </dt>
          <dd className="mt-0.5 text-sm text-text">
            <TimeAgo iso={track.createdAt} />
          </dd>
        </div>

        {isOwner && (
          <div className="col-span-2">
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Visibility
            </dt>
            <dd className="mt-0.5 text-sm text-text">
              <ToggleGroup
                options={[
                  { value: "private", label: "Private" },
                  { value: "public", label: "Public" },
                ]}
                value={track.public ? "public" : "private"}
                onChange={(v) => {
                  const next = v === "public"
                  if (next !== (track.public ?? false)) {
                    saveField({ public: next })
                  }
                }}
                ariaLabel="Track visibility"
                className="w-fit"
              />
            </dd>
          </div>
        )}

        {track.author && (
          <div className="col-span-2">
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Author
            </dt>
            <dd className="mt-0.5 text-sm text-text">
              {track.authorLinkUrl ? (
                <a
                  href={externalUrl(track.authorLinkUrl)}
                  className="text-text-secondary hover:text-text underline transition-colors"
                >
                  {track.author}
                </a>
              ) : (
                track.author
              )}
            </dd>
          </div>
        )}

        {track.linkUrl && (
          <div className="col-span-2">
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Link
            </dt>
            <dd className="mt-0.5 text-sm truncate">
              <a
                href={externalUrl(track.linkUrl)}
                className="text-text-secondary hover:text-text underline text-sm transition-colors"
              >
                {track.linkUrl}
              </a>
            </dd>
          </div>
        )}
      </dl>

      {(track.tags.length > 0 || isOwner) && (
        <div className="mt-3">
          {editingField === "tags" ? (
            <div>
              <TagsInput value={editTags} onChange={setEditTags} />
              <div className="mt-2 flex gap-2">
                <Button
                  variant="secondary"
                  onClick={() => saveField({ tags: editTags })}
                  disabled={editMutation.isPending}
                >
                  {editMutation.isPending ? "Saving..." : "Save"}
                </Button>
                <Button variant="ghost" onClick={() => setEditingField(null)}>
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <div
              className={`flex flex-wrap items-center gap-1.5 ${isOwner ? "editable-field pb-0.5" : ""}`}
              onClick={() => {
                if (isOwner) {
                  setEditTags([...track.tags])
                  setEditingField("tags")
                }
              }}
              role={isOwner ? "button" : undefined}
              tabIndex={isOwner ? 0 : undefined}
              onKeyDown={(e) => {
                if (isOwner && e.key === "Enter") {
                  setEditTags([...track.tags])
                  setEditingField("tags")
                }
              }}
              aria-label={isOwner ? "Edit tags" : undefined}
            >
              {track.tags.length > 0 ? (
                track.tags.map((tag) => <Badge key={tag}>{tag}</Badge>)
              ) : (
                <span className="text-sm text-text-muted hover:text-primary transition-colors">
                  Add tags...
                </span>
              )}
            </div>
          )}
        </div>
      )}

      {track.similarTracks.length > 0 && (
        <div className="mt-3">
          <SectionHeading>Similar tracks</SectionHeading>
          <ul className="mt-1 flex flex-wrap gap-x-4 gap-y-0.5">
            {track.similarTracks.map((st) => (
              <li key={st.uuid}>
                <Link
                  to={`/tracks/${st.uuid}`}
                  className="text-sm text-text-secondary hover:text-text transition-colors"
                >
                  {st.name}
                  <span className="ml-1 text-text-muted">
                    {formatDistance(st.totalDistanceM)}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="mt-3 flex items-center justify-between">
        <a
          href={`/api/tracks/${track.uuid}/download`}
          className="text-sm text-text-muted hover:text-text-secondary transition-colors"
        >
          Download original file
        </a>
        {isOwner && (
          <div className="flex items-center gap-2">
            {confirmDelete ? (
              <>
                <span className="text-sm text-text-secondary">
                  Delete this track?
                </span>
                <Button
                  variant="danger"
                  onClick={handleDelete}
                  disabled={deleteMutation.isPending}
                >
                  {deleteMutation.isPending ? "Deleting..." : "Confirm"}
                </Button>
                <Button variant="ghost" onClick={() => setConfirmDelete(false)}>
                  Cancel
                </Button>
              </>
            ) : (
              <Button variant="ghost" onClick={() => setConfirmDelete(true)}>
                Delete
              </Button>
            )}
          </div>
        )}
      </div>
    </>
  )
}
