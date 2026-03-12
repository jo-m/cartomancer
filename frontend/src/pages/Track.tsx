import { Link, useParams } from "react-router-dom"
import { useQueryClient } from "@tanstack/react-query"
import { $api } from "../api/client"
import { useSession } from "../context/SessionContext"
import StarIcon from "../assets/StarIcon"
import { SPORT_LABELS, SUB_SPORT_LABELS } from "../lib/sports"

const TRACK_TYPE_LABELS: Record<number, string> = {
  0: "Unknown",
  1: "Planned",
  2: "Recorded",
}

const FILE_FORMAT_LABELS: Record<number, string> = {
  0: "GPX",
  1: "FIT",
}

function formatDistance(m: number): string {
  return `${(m / 1000).toFixed(1)} km`
}

function formatAscent(m: number): string {
  return `${Math.round(m)} m`
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  })
}

export default function Track() {
  const { uuid } = useParams<{ uuid: string }>()
  const { user } = useSession()
  const queryClient = useQueryClient()

  const { data, isLoading, error } = $api.useQuery("get", "/tracks/{uuid}", {
    params: { path: { uuid: uuid! } },
  })

  const starMutation = $api.useMutation("post", "/tracks/{uuid}/star")
  const unstarMutation = $api.useMutation("delete", "/tracks/{uuid}/star")

  async function toggleStar() {
    if (!data) return
    if (data.isStarred) {
      await unstarMutation.mutateAsync({
        params: { path: { uuid: data.uuid } },
      })
    } else {
      await starMutation.mutateAsync({ params: { path: { uuid: data.uuid } } })
    }
    await queryClient.invalidateQueries({
      queryKey: ["get", "/tracks/{uuid}"],
    })
  }

  if (isLoading) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-10">
        <p className="text-gray-500">Loading…</p>
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-10">
        <p className="text-red-600">
          {(error as Error | null)?.message ?? "Track not found."}
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-10">
      <Link to="/" className="text-sm text-gray-500 hover:text-gray-700">
        ← Tracks
      </Link>

      <div className="mt-4 flex items-start justify-between gap-4">
        <h1 className="text-2xl font-bold text-gray-900">{data.name}</h1>
        {user && (
          <button
            onClick={toggleStar}
            disabled={starMutation.isPending || unstarMutation.isPending}
            className="shrink-0 cursor-pointer rounded border border-gray-200 p-1.5 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <StarIcon
              className={`h-5 w-5 ${data.isStarred ? "text-yellow-400" : "text-gray-300"}`}
            />
          </button>
        )}
      </div>

      {data.description && (
        <p className="mt-2 text-sm text-gray-600">{data.description}</p>
      )}

      <div className="mt-6 overflow-hidden rounded-lg border border-gray-200 bg-gray-50">
        <img
          src={`/api/tracks/${data.uuid}/preview.svg?size=512`}
          alt="Track preview"
          className="w-full object-contain"
        />
      </div>

      <div className="mt-4 overflow-hidden rounded-lg border border-gray-200 bg-gray-50">
        <img
          src={`/api/tracks/${data.uuid}/profile.svg?size=512`}
          alt="Elevation profile"
          className="w-full"
        />
      </div>

      <dl className="mt-6 grid grid-cols-2 gap-x-8 gap-y-4 sm:grid-cols-3">
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Distance
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {formatDistance(data.totalDistanceM)}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Ascent
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {formatAscent(data.totalAscentM)}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Sport
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {SPORT_LABELS[data.sport] ?? data.sport}
            {data.subSport !== 0 && (
              <span className="ml-1 text-gray-500">
                ({SUB_SPORT_LABELS[data.subSport] ?? data.subSport})
              </span>
            )}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Type
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {TRACK_TYPE_LABELS[data.trackType] ?? data.trackType}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Format
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {FILE_FORMAT_LABELS[data.fileFormat] ?? data.fileFormat}
          </dd>
        </div>
        {data.originalCreatedAt && (
          <div>
            <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
              Activity date
            </dt>
            <dd className="mt-1 text-sm text-gray-900">
              {formatDate(data.originalCreatedAt)}
            </dd>
          </div>
        )}
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Uploaded
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {formatDate(data.createdAt)}
          </dd>
        </div>
        {data.source && (
          <div className="col-span-2 sm:col-span-3">
            <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
              Source
            </dt>
            <dd className="mt-1 text-sm text-gray-900">{data.source}</dd>
          </div>
        )}
      </dl>

      {data.tags.length > 0 && (
        <div className="mt-6">
          <p className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Tags
          </p>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {data.tags.map((tag) => (
              <span
                key={tag}
                className="rounded-full border border-gray-200 bg-gray-100 px-2.5 py-0.5 text-xs text-gray-700"
              >
                {tag}
              </span>
            ))}
          </div>
        </div>
      )}

      <div className="mt-6">
        <a
          href={`/api/tracks/${data.uuid}/download`}
          className="text-sm text-gray-500 hover:text-gray-700"
        >
          Download original file
        </a>
      </div>
    </div>
  )
}
