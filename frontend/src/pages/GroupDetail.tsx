import { Link, useParams } from "react-router-dom"
import { $api } from "../api/client"

function formatDistance(m: number): string {
  return `${(m / 1000).toFixed(1)} km`
}

function formatAscent(m: number): string {
  return `${Math.round(m)} m`
}

/** GroupDetail displays all member tracks of a single track group. */
export default function GroupDetail() {
  const { uuid } = useParams<{ uuid: string }>()
  const { data, isLoading, error } = $api.useQuery(
    "get",
    "/tracks/groups/{uuid}",
    {
      params: { path: { uuid: uuid! } },
    }
  )

  return (
    <div className="mx-auto max-w-5xl px-4 py-6">
      <Link
        to="/tracks/groups"
        className="text-sm text-gray-500 hover:text-gray-700"
      >
        &larr; All groups
      </Link>

      {isLoading && <p className="mt-6 text-sm text-gray-500">Loading...</p>}
      {error && (
        <p className="mt-6 text-sm text-red-600">
          {(error as unknown as Error).message}
        </p>
      )}

      {data && (
        <>
          <h1 className="mt-4 text-lg font-semibold text-gray-900">
            Group ({data.tracks.length} tracks)
          </h1>

          {data.tracks.length === 0 ? (
            <p className="mt-6 text-sm text-gray-500">
              No tracks in this group.
            </p>
          ) : (
            <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
              {data.tracks.map((track) => (
                <Link
                  key={track.uuid}
                  to={`/tracks/${track.uuid}`}
                  className="group relative block rounded-lg border border-gray-200 bg-white hover:border-gray-400"
                >
                  <div className="aspect-square overflow-hidden rounded-t-lg bg-gray-50">
                    <img
                      src={`/api/tracks/${track.uuid}/preview.svg?size=256`}
                      alt="Track preview"
                      className="h-full w-full object-contain"
                    />
                  </div>
                  <div className="p-2.5">
                    <div className="flex items-center gap-1.5">
                      <img
                        src={`/api/users/${track.userUuid}/avatar`}
                        alt=""
                        className="h-4 w-4 shrink-0 rounded-full"
                      />
                      <p className="truncate text-sm font-medium text-gray-900">
                        {track.name}
                      </p>
                    </div>
                    <p className="mt-0.5 text-xs text-gray-500">
                      {track.userName} &middot;{" "}
                      {formatDistance(track.totalDistanceM)} &middot;{" "}
                      {formatAscent(track.totalAscentM)}
                    </p>
                    <div className="mt-1.5 overflow-hidden rounded bg-gray-50">
                      <img
                        src={`/api/tracks/${track.uuid}/profile.svg?size=256`}
                        alt="Elevation profile"
                        className="w-full"
                      />
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
