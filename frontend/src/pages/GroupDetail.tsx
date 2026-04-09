import { Link, useParams } from "react-router-dom"
import { $api } from "../api/client"
import TrackCard from "../components/TrackCard"
import PageContainer from "../components/ui/PageContainer"

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
    <PageContainer>
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
            <div className="mt-4 grid grid-cols-2 gap-6 sm:grid-cols-3 lg:grid-cols-4">
              {data.tracks.map((track, index) => (
                <TrackCard
                  key={track.uuid}
                  track={track}
                  index={index}
                  isSelected={false}
                  selectionActive={false}
                  canSelect={false}
                  showStar={false}
                  onToggleStar={() => {}}
                  onSelect={() => {}}
                />
              ))}
            </div>
          )}
        </>
      )}
    </PageContainer>
  )
}
