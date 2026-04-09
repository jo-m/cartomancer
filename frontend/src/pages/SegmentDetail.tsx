import { Link, useParams } from "react-router-dom"
import { $api } from "../api/client"
import SegmentMap from "../components/SegmentMap"
import PageContainer from "../components/ui/PageContainer"
import Card from "../components/ui/Card"

/** SegmentDetail shows details for a single segment. */
export default function SegmentDetail() {
  const { uuid } = useParams<{ uuid: string }>()

  const { data, isLoading, error } = $api.useQuery(
    "get",
    "/admin/segments/{uuid}",
    {
      params: { path: { uuid: uuid! } },
    }
  )

  return (
    <PageContainer>
      <Link
        to="/admin/segments"
        className="text-sm text-text-muted hover:text-text-secondary transition-colors"
      >
        &larr; All segments
      </Link>

      {isLoading && <p className="mt-6 text-sm text-text-muted">Loading...</p>}
      {error && (
        <p role="alert" className="mt-6 text-sm text-error">
          {error.message}
        </p>
      )}

      {data && (
        <div className="mt-4">
          <h1 className="text-lg font-semibold text-text">
            Segment: {(data.distanceM / 1000).toFixed(1)} km
          </h1>

          <dl className="mt-4 grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
            <div>
              <dt className="text-text-muted">Distance</dt>
              <dd className="font-medium text-text">
                {(data.distanceM / 1000).toFixed(2)} km
              </dd>
            </div>
            <div>
              <dt className="text-text-muted">Tracks</dt>
              <dd className="font-medium text-text">{data.nTracks}</dd>
            </div>
            <div>
              <dt className="text-text-muted">H3 Resolution</dt>
              <dd className="font-medium text-text">{data.h3Resolution}</dd>
            </div>
            <div>
              <dt className="text-text-muted">Ascent</dt>
              <dd className="font-medium text-text">
                {data.ascentM.toFixed(0)} m
              </dd>
            </div>
          </dl>

          <div className="mt-6">
            <SegmentMap polyline={data.polyline} />
          </div>

          <div className="mt-6">
            <h2 className="text-sm font-medium text-text">Junctions</h2>
            <div className="mt-2 grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Card className="px-4 py-3">
                <p className="text-xs text-text-muted">Start</p>
                <p className="text-sm text-text">
                  {data.startJunction.lat.toFixed(5)},{" "}
                  {data.startJunction.lon.toFixed(5)}
                </p>
                <p className="mt-0.5 text-xs text-text-muted">
                  {data.startJunction.h3Cell}
                </p>
              </Card>
              <Card className="px-4 py-3">
                <p className="text-xs text-text-muted">End</p>
                <p className="text-sm text-text">
                  {data.endJunction.lat.toFixed(5)},{" "}
                  {data.endJunction.lon.toFixed(5)}
                </p>
                <p className="mt-0.5 text-xs text-text-muted">
                  {data.endJunction.h3Cell}
                </p>
              </Card>
            </div>
          </div>

          <div className="mt-6">
            <h2 className="text-sm font-medium text-text">
              Tracks ({data.trackUuids.length})
            </h2>
            <ul className="mt-2 space-y-1">
              {data.trackUuids.map((trackUuid) => (
                <li key={trackUuid}>
                  <Link
                    to={`/tracks/${trackUuid}`}
                    className="text-sm text-primary hover:text-primary-hover transition-colors"
                  >
                    {trackUuid}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </PageContainer>
  )
}
