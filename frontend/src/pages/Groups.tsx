import { Link } from "react-router-dom"
import { $api } from "../api/client"
import useDocumentTitle from "../hooks/useDocumentTitle"
import PageContainer from "../components/ui/PageContainer"

/** Groups lists all track groups for the current user. */
export default function Groups() {
  useDocumentTitle("Groups")
  const { data, isLoading, error } = $api.useQuery("get", "/tracks/groups")

  return (
    <PageContainer>
      <h1 className="text-lg font-semibold text-text">Groups</h1>
      <p className="mt-1 text-sm text-text-muted">
        Tracks grouped by similarity. New uploads may take a few minutes to
        appear in groups.
      </p>

      {isLoading && <p className="mt-6 text-sm text-text-muted">Loading...</p>}
      {error && (
        <p role="alert" className="mt-6 text-sm text-error">
          {error.message}
        </p>
      )}

      {data && data.groups.length === 0 && (
        <p className="mt-6 text-sm text-text-muted">No groups found.</p>
      )}

      {data && data.groups.length > 0 && (
        <ul className="mt-4 space-y-2">
          {data.groups.map((g) => (
            <li key={g.uuid}>
              <Link
                to={`/tracks/groups/${g.uuid}`}
                className="block rounded-lg border border-border bg-panel px-4 py-3 hover:border-border-hover transition-colors"
              >
                <p className="text-sm font-medium text-text">{g.sampleName}</p>
                <p className="mt-0.5 text-xs text-text-muted">
                  {g.memberCount} tracks
                </p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </PageContainer>
  )
}
