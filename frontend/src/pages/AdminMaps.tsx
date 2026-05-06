import { $api } from "../api/client"
import useDocumentTitle from "../hooks/useDocumentTitle"
import useToast from "../hooks/useToast"
import PageContainer from "../components/ui/PageContainer"
import Card from "../components/ui/Card"
import Button from "../components/ui/Button"
import Toast from "../components/Toast"
import CopyIdCell from "../components/CopyIdCell"
import AdminTabs from "../components/admin/AdminTabs"
import AdminCard, {
  AdminCardField,
  AdminCardFooter,
  AdminCardHeader,
} from "../components/admin/AdminCard"
import TimeAgo from "../components/TimeAgo"

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

  const {
    data: builds,
    isLoading,
    refetch,
  } = $api.useQuery("get", "/admin/maps", {})
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

  function renderStatus(b: (typeof rows)[0]) {
    if (b.markedForDeletion) {
      return <span className="text-xs text-error">Pending deletion</span>
    }
    if (b.ready) {
      return <span className="text-xs text-success">Ready</span>
    }
    return <span className="text-xs text-text-muted">Not ready</span>
  }

  return (
    <PageContainer size="2xl">
      <AdminTabs current="maps" />

      <Card className="hidden overflow-x-auto md:block">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs font-medium text-text-muted">
              <th className="px-4 py-3">ID</th>
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
                <td className="px-4 py-3">
                  <CopyIdCell
                    id={b.uuid}
                    onCopied={() => showToast("Copied to clipboard", "success")}
                  />
                </td>
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
                <td className="px-4 py-3">{renderStatus(b)}</td>
                <td className="px-4 py-3 text-text-muted">
                  <TimeAgo iso={b.createdAt} />
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
                  colSpan={9}
                  className="px-4 py-6 text-center text-sm text-text-muted"
                >
                  {isLoading ? "Loading..." : "No map builds found."}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      <div className="space-y-3 md:hidden">
        {rows.map((b) => (
          <AdminCard
            key={b.uuid}
            className={b.markedForDeletion ? "opacity-50" : ""}
          >
            <AdminCardHeader>
              <span className="min-w-0 break-all font-mono text-xs text-text">
                {b.key}
              </span>
              <CopyIdCell
                id={b.uuid}
                onCopied={() => showToast("Copied to clipboard", "success")}
              />
            </AdminCardHeader>
            <AdminCardField label="Version">{b.version}</AdminCardField>
            <AdminCardField label="Zoom">{b.maxZoom}</AdminCardField>
            <AdminCardField label="Bbox">
              {formatBbox(
                b.bboxMinLon,
                b.bboxMinLat,
                b.bboxMaxLon,
                b.bboxMaxLat
              )}
            </AdminCardField>
            <AdminCardField label="Size">
              {b.localSize != null ? formatBytes(b.localSize) : "--"}
            </AdminCardField>
            <AdminCardField label="Status">{renderStatus(b)}</AdminCardField>
            <AdminCardField label="Created">
              <TimeAgo iso={b.createdAt} />
            </AdminCardField>
            {!b.markedForDeletion && (
              <AdminCardFooter>
                <Button
                  variant="danger"
                  onClick={() => handleMarkForDeletion(b.uuid, b.key)}
                  disabled={markMutation.isPending}
                  title="Mark for deletion on next cleanup"
                >
                  Delete
                </Button>
              </AdminCardFooter>
            )}
          </AdminCard>
        ))}
        {rows.length === 0 && (
          <Card className="px-4 py-6 text-center text-sm text-text-muted">
            {isLoading ? "Loading..." : "No map builds found."}
          </Card>
        )}
      </div>

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
