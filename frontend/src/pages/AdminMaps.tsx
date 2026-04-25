import { Link } from "react-router-dom"
import { $api } from "../api/client"
import useDocumentTitle from "../hooks/useDocumentTitle"
import useToast from "../hooks/useToast"
import PageContainer from "../components/ui/PageContainer"
import Card from "../components/ui/Card"
import Button from "../components/ui/Button"
import Toast from "../components/Toast"

/** Formats a byte count into a human-readable size string. */
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024)
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

function formatBbox(
  minLon?: number,
  minLat?: number,
  maxLon?: number,
  maxLat?: number
): string {
  if (minLon == null) return "World"
  return `${minLat?.toFixed(1)}, ${minLon?.toFixed(1)} — ${maxLat?.toFixed(1)}, ${maxLon?.toFixed(1)}`
}

export default function AdminMaps() {
  useDocumentTitle("Maps")
  const { toast, showToast, dismissToast } = useToast()

  const { data: builds, refetch } = $api.useQuery("get", "/admin/maps", {})
  const markMutation = $api.useMutation(
    "post",
    "/admin/maps/{uuid}/mark-for-deletion"
  )

  async function handleMarkForDeletion(uuid: string, key: string) {
    if (
      !window.confirm(
        `Mark ${key} for deletion? The file will be removed on the next cleanup run.`
      )
    )
      return
    try {
      await markMutation.mutateAsync({ params: { path: { uuid } } })
      showToast(`Marked ${key} for deletion`, "success")
      refetch()
    } catch (err) {
      showToast((err as Error).message)
    }
  }

  const rows = builds ?? []

  return (
    <PageContainer size="lg">
      <div className="mb-6 flex items-center gap-4">
        <h1 className="text-2xl font-semibold text-text">Admin</h1>
        <Link
          to="/admin/users"
          className="pb-0.5 text-sm font-medium text-text-muted hover:text-text-secondary transition-colors"
        >
          Users
        </Link>
        <Link
          to="/admin/forecasts"
          className="pb-0.5 text-sm font-medium text-text-muted hover:text-text-secondary transition-colors"
        >
          Forecasts
        </Link>
        <Link
          to="/admin/maps"
          className="border-b-2 border-primary pb-0.5 text-sm font-medium text-text"
          aria-current="page"
        >
          Maps
        </Link>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs font-medium text-text-muted">
              <th className="px-4 py-3">Build key</th>
              <th className="px-4 py-3">Version</th>
              <th className="px-4 py-3">Zoom</th>
              <th className="px-4 py-3">Bbox</th>
              <th className="px-4 py-3">Size</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Created</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            {rows.map((b) => (
              <tr
                key={b.uuid}
                className={`border-b border-border last:border-0 ${b.markedForDeletion ? "opacity-50" : ""}`}
              >
                <td className="px-4 py-3 font-mono text-xs text-text">
                  {b.key}
                </td>
                <td className="px-4 py-3 text-text-secondary">{b.version}</td>
                <td className="px-4 py-3 text-text-secondary">{b.maxZoom}</td>
                <td className="px-4 py-3 text-xs text-text-muted">
                  {formatBbox(
                    b.bboxMinLon,
                    b.bboxMinLat,
                    b.bboxMaxLon,
                    b.bboxMaxLat
                  )}
                </td>
                <td className="px-4 py-3 text-text-muted">
                  {b.localSize != null ? formatBytes(b.localSize) : "--"}
                </td>
                <td className="px-4 py-3">
                  {b.markedForDeletion ? (
                    <span className="text-xs text-error">Pending deletion</span>
                  ) : b.ready ? (
                    <span className="text-xs text-success">Ready</span>
                  ) : (
                    <span className="text-xs text-text-muted">Not ready</span>
                  )}
                </td>
                <td className="px-4 py-3 text-text-muted">
                  {b.createdAt.slice(0, 10)}
                </td>
                <td className="px-4 py-3">
                  {!b.markedForDeletion && (
                    <Button
                      variant="danger"
                      onClick={() => handleMarkForDeletion(b.uuid, b.key)}
                      disabled={markMutation.isPending}
                      title="Mark for deletion on next cleanup"
                    >
                      Delete
                    </Button>
                  )}
                </td>
              </tr>
            ))}
            {rows.length === 0 && (
              <tr>
                <td
                  colSpan={8}
                  className="px-4 py-6 text-center text-sm text-text-muted"
                >
                  No map builds found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {toast && (
        <Toast
          key={toast.key}
          message={toast.message}
          variant={toast.variant}
          onDismiss={dismissToast}
        />
      )}
    </PageContainer>
  )
}
