import { Link, useParams } from "react-router-dom"
import { $api } from "../api/client"
import SvgPreview from "../components/SvgPreview"
import PageContainer from "../components/ui/PageContainer"
import { formatDistance, formatAscent } from "../lib/format"

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
    <PageContainer className="py-6">
      <Link
        to="/tracks/groups"
        className="text-sm text-text-muted hover:text-text-secondary transition-colors"
      >
        &larr; All groups
      </Link>

      {isLoading && <p className="mt-6 text-sm text-text-muted">Loading...</p>}
      {error && (
        <p role="alert" className="mt-6 text-sm text-error">
          {error.message}
        </p>
      )}

      {data && (
        <>
          <h1 className="mt-4 text-lg font-semibold text-text">
            Group ({data.tracks.length} tracks)
          </h1>

          {data.tracks.length === 0 ? (
            <p className="mt-6 text-sm text-text-muted">
              No tracks in this group.
            </p>
          ) : (
            <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
              {data.tracks.map((track) => (
                <Link
                  key={track.uuid}
                  to={`/tracks/${track.uuid}`}
                  className="group relative block rounded-lg border border-border bg-panel hover:border-border-hover transition-colors"
                >
                  <div className="aspect-square overflow-hidden rounded-t-lg bg-surface text-track">
                    <SvgPreview
                      src={`/api/tracks/${track.uuid}/preview.svg?size=256`}
                      alt="Track preview"
                      className="h-full w-full object-contain"
                    />
                  </div>
                  <div className="p-2.5">
                    <div className="flex items-center gap-1.5">
                      <img
                        src={`/api/users/${track.user.uuid}/avatar`}
                        alt=""
                        className="h-4 w-4 shrink-0 rounded-full"
                      />
                      <p className="truncate text-sm font-medium text-text">
                        {track.name}
                      </p>
                    </div>
                    <p className="mt-0.5 text-xs text-text-muted">
                      {track.user.name} &middot;{" "}
                      {formatDistance(track.totalDistanceM)} &middot;{" "}
                      {formatAscent(track.totalAscentM)}
                    </p>
                    <div className="mt-1.5 overflow-hidden rounded bg-surface text-track">
                      <SvgPreview
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
    </PageContainer>
  )
}
