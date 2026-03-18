import { Link } from "react-router-dom"
import { $api } from "../api/client"

/** Groups lists all track groups for the current user. */
export default function Groups() {
  const { data, isLoading, error } = $api.useQuery("get", "/tracks/groups")

  return (
    <div className="mx-auto max-w-5xl px-4 py-6">
      <h1 className="text-lg font-semibold text-gray-900">Groups</h1>
      <p className="mt-1 text-sm text-gray-500">
        Tracks grouped by similarity.
      </p>

      {isLoading && <p className="mt-6 text-sm text-gray-500">Loading...</p>}
      {error && (
        <p className="mt-6 text-sm text-red-600">
          {(error as unknown as Error).message}
        </p>
      )}

      {data && data.groups.length === 0 && (
        <p className="mt-6 text-sm text-gray-500">No groups found.</p>
      )}

      {data && data.groups.length > 0 && (
        <ul className="mt-4 space-y-2">
          {data.groups.map((g) => (
            <li key={g.uuid}>
              <Link
                to={`/tracks/groups/${g.uuid}`}
                className="block rounded-lg border border-gray-200 bg-white px-4 py-3 hover:border-gray-400"
              >
                <p className="text-sm font-medium text-gray-900">
                  {g.sampleName}
                </p>
                <p className="mt-0.5 text-xs text-gray-500">
                  {g.memberCount} tracks
                </p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
