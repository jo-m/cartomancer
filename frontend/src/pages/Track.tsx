import { Link, useParams } from "react-router-dom"
import { $api } from "../api/client"
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

  const { data, isLoading, error } = $api.useQuery("get", "/tracks/{uuid}", {
    params: { path: { uuid: uuid! } },
  })

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
      <Link to="/tracks" className="text-sm text-gray-500 hover:text-gray-700">
        ← Tracks
      </Link>

      <h1 className="mt-4 text-2xl font-bold text-gray-900">{data.name}</h1>

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
