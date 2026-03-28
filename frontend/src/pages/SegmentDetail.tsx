import { Link, useParams } from "react-router-dom"
import { $api } from "../api/client"
import SegmentMap from "../components/SegmentMap"

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
    <div className="mx-auto max-w-5xl px-4 py-6">
      <Link
        to="/admin/segments"
        className="text-sm text-gray-500 hover:text-gray-700"
      >
        &larr; All segments
      </Link>

      {isLoading && <p className="mt-6 text-sm text-gray-500">Loading...</p>}
      {error && (
        <p className="mt-6 text-sm text-red-600">
          {(error as unknown as Error).message}
        </p>
      )}

      {data && (
        <div className="mt-4">
          <h1 className="text-lg font-semibold text-gray-900">
            Segment: {(data.distanceM / 1000).toFixed(1)} km
          </h1>

          <dl className="mt-4 grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
            <div>
              <dt className="text-gray-500">Distance</dt>
              <dd className="font-medium text-gray-900">
                {(data.distanceM / 1000).toFixed(2)} km
              </dd>
            </div>
            <div>
              <dt className="text-gray-500">Tracks</dt>
              <dd className="font-medium text-gray-900">{data.nTracks}</dd>
            </div>
            <div>
              <dt className="text-gray-500">H3 Resolution</dt>
              <dd className="font-medium text-gray-900">{data.h3Resolution}</dd>
            </div>
            <div>
              <dt className="text-gray-500">Ascent</dt>
              <dd className="font-medium text-gray-900">
                {data.ascentM.toFixed(0)} m
              </dd>
            </div>
          </dl>

          <div className="mt-6">
            <SegmentMap polyline={data.polyline} />
          </div>

          <div className="mt-6">
            <h2 className="text-sm font-medium text-gray-900">Junctions</h2>
            <div className="mt-2 grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="rounded-lg border border-gray-200 bg-white px-4 py-3">
                <p className="text-xs text-gray-500">Start</p>
                <p className="text-sm text-gray-900">
                  {data.startJunction.lat.toFixed(5)},{" "}
                  {data.startJunction.lon.toFixed(5)}
                </p>
                <p className="mt-0.5 text-xs text-gray-400">
                  {data.startJunction.h3Cell}
                </p>
              </div>
              <div className="rounded-lg border border-gray-200 bg-white px-4 py-3">
                <p className="text-xs text-gray-500">End</p>
                <p className="text-sm text-gray-900">
                  {data.endJunction.lat.toFixed(5)},{" "}
                  {data.endJunction.lon.toFixed(5)}
                </p>
                <p className="mt-0.5 text-xs text-gray-400">
                  {data.endJunction.h3Cell}
                </p>
              </div>
            </div>
          </div>

          <div className="mt-6">
            <h2 className="text-sm font-medium text-gray-900">
              Tracks ({data.trackUuids.length})
            </h2>
            <ul className="mt-2 space-y-1">
              {data.trackUuids.map((trackUuid) => (
                <li key={trackUuid}>
                  <Link
                    to={`/tracks/${trackUuid}`}
                    className="text-sm text-blue-600 hover:text-blue-800"
                  >
                    {trackUuid}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </div>
  )
}
