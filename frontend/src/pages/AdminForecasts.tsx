import { useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { $api } from "../api/client"
import useDocumentTitle from "../hooks/useDocumentTitle"
import useToast from "../hooks/useToast"
import { useUrlState, stringParam } from "../hooks/useUrlState"
import PageContainer from "../components/ui/PageContainer"
import Card from "../components/ui/Card"
import Input from "../components/ui/Input"
import Toast from "../components/Toast"
import CopyIdCell from "../components/CopyIdCell"

/** Formats a byte count into a human-readable size string. */
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export default function AdminForecasts() {
  useDocumentTitle("Forecasts")
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const { toast, showToast, dismissToast } = useToast()
  const searchSchema = useMemo(() => ({ q: stringParam() }), [])
  const [urlState, setUrlState] = useUrlState(searchSchema)
  const search = urlState.q
  const setSearch = (v: string) => setUrlState({ q: v })

  const { data } = $api.useQuery("get", "/admin/forecasts", {})
  const forecasts = data?.forecasts ?? []

  const filtered = forecasts.filter(
    (f) =>
      f.attribution.text.toLowerCase().includes(search.toLowerCase()) ||
      f.referenceTime.includes(search)
  )

  return (
    <PageContainer size="2xl">
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
          className="border-b-2 border-primary pb-0.5 text-sm font-medium text-text"
          aria-current="page"
        >
          Forecasts
        </Link>
        <Link
          to="/admin/maps"
          className="pb-0.5 text-sm font-medium text-text-muted hover:text-text-secondary transition-colors"
        >
          Maps
        </Link>
      </div>

      <div className="mb-4">
        <Input
          type="text"
          placeholder="Search forecasts..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          aria-label="Search forecasts"
          className="max-w-xs"
        />
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs font-medium text-text-muted">
              <th className="px-4 py-3">ID</th>
              <th className="px-4 py-3">Reference time</th>
              <th className="px-4 py-3">Attribution</th>
              <th className="px-4 py-3">Bounds</th>
              <th className="px-4 py-3">Files</th>
              <th className="px-4 py-3">Created</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((f) => {
              const totalSize = f.files.reduce(
                (sum, file) => sum + file.fileSize,
                0
              )
              const isExpanded = expandedId === f.id

              return (
                <tr key={f.id} className="border-b border-border last:border-0">
                  <td className="px-4 py-3 align-top">
                    <CopyIdCell
                      id={f.id}
                      onCopied={() =>
                        showToast("Copied to clipboard", "success")
                      }
                    />
                  </td>
                  <td colSpan={5} className="px-0 py-0">
                    <button
                      onClick={() => setExpandedId(isExpanded ? null : f.id)}
                      className="flex w-full cursor-pointer items-center text-left"
                    >
                      <span className="w-1/5 px-4 py-3 text-text">
                        {f.referenceTime.slice(0, 16).replace("T", " ")}
                      </span>
                      <span className="w-1/5 px-4 py-3 text-text-secondary">
                        <a
                          href={f.attribution.href}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="hover:underline"
                          onClick={(e) => e.stopPropagation()}
                        >
                          {f.attribution.text}
                        </a>
                      </span>
                      <span className="w-1/5 px-4 py-3 text-xs text-text-muted">
                        {f.bounds
                          ? `${f.bounds.min.lat.toFixed(1)}, ${f.bounds.min.lon.toFixed(1)} - ${f.bounds.max.lat.toFixed(1)}, ${f.bounds.max.lon.toFixed(1)}`
                          : "--"}
                      </span>
                      <span className="w-1/5 px-4 py-3 text-text-secondary">
                        {f.files.length} ({formatBytes(totalSize)})
                      </span>
                      <span className="w-1/5 px-4 py-3 text-text-muted">
                        {f.createdAt.slice(0, 10)}
                      </span>
                    </button>

                    {isExpanded && f.files.length > 0 && (
                      <div className="border-t border-border bg-surface px-4 py-3">
                        <table className="w-full text-xs">
                          <thead>
                            <tr className="text-text-muted">
                              <th className="pb-1 pr-4 text-left font-medium">
                                ID
                              </th>
                              <th className="pb-1 pr-4 text-left font-medium">
                                Variable
                              </th>
                              <th className="pb-1 pr-4 text-left font-medium">
                                Valid time
                              </th>
                              <th className="pb-1 pr-4 text-left font-medium">
                                Valid until
                              </th>
                              <th className="pb-1 text-left font-medium">
                                Size
                              </th>
                            </tr>
                          </thead>
                          <tbody>
                            {f.files.map((file) => (
                              <tr key={file.id}>
                                <td className="py-0.5 pr-4">
                                  <CopyIdCell
                                    id={file.id}
                                    onCopied={() =>
                                      showToast(
                                        "Copied to clipboard",
                                        "success"
                                      )
                                    }
                                  />
                                </td>
                                <td className="py-0.5 pr-4 text-text-secondary">
                                  {file.variable}
                                </td>
                                <td className="py-0.5 pr-4 text-text-muted">
                                  {file.validTime
                                    .slice(0, 16)
                                    .replace("T", " ")}
                                </td>
                                <td className="py-0.5 pr-4 text-text-muted">
                                  {file.validUntilTime
                                    .slice(0, 16)
                                    .replace("T", " ")}
                                </td>
                                <td className="py-0.5 text-text-muted">
                                  {formatBytes(file.fileSize)}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </td>
                </tr>
              )
            })}
            {filtered.length === 0 && (
              <tr>
                <td
                  colSpan={6}
                  className="px-4 py-6 text-center text-sm text-text-muted"
                >
                  No forecasts found.
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
