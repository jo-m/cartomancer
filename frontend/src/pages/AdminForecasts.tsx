import { useMemo, useState } from "react"
import { $api } from "../api/client"
import useDocumentTitle from "../hooks/useDocumentTitle"
import useToast from "../hooks/useToast"
import { useUrlState, stringParam } from "../hooks/useUrlState"
import PageContainer from "../components/ui/PageContainer"
import Card from "../components/ui/Card"
import Input from "../components/ui/Input"
import Toast from "../components/Toast"
import CopyIdCell from "../components/CopyIdCell"
import AdminTabs from "../components/admin/AdminTabs"
import AdminCard, {
  AdminCardField,
  AdminCardFooter,
  AdminCardHeader,
} from "../components/admin/AdminCard"
import TimeAgo from "../components/TimeAgo"
import { fmtAbsolute, fmtDateTime } from "../lib/time"

/** Formats a byte count into a human-readable size string. */
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatBounds(
  bounds:
    | { min: { lat: number; lon: number }; max: { lat: number; lon: number } }
    | null
    | undefined
) {
  if (!bounds) return "--"
  return `${bounds.min.lat.toFixed(1)}, ${bounds.min.lon.toFixed(1)} - ${bounds.max.lat.toFixed(1)}, ${bounds.max.lon.toFixed(1)}`
}

export default function AdminForecasts() {
  useDocumentTitle("Forecasts")
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const { toast, showToast, dismissToast } = useToast()
  const searchSchema = useMemo(() => ({ q: stringParam() }), [])
  const [urlState, setUrlState] = useUrlState(searchSchema)
  const search = urlState.q
  const setSearch = (v: string) => setUrlState({ q: v })

  const { data, isLoading } = $api.useQuery("get", "/admin/forecasts", {})
  const forecasts = data?.forecasts ?? []

  const filtered = forecasts.filter(
    (f) =>
      f.attribution.text.toLowerCase().includes(search.toLowerCase()) ||
      f.referenceTime.includes(search)
  )

  function copyToast() {
    showToast("Copied to clipboard", "success")
  }

  return (
    <PageContainer size="2xl">
      <AdminTabs current="forecasts" />

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

      <Card className="hidden overflow-x-auto md:block">
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
                    <CopyIdCell id={f.id} onCopied={copyToast} />
                  </td>
                  <td colSpan={5} className="px-0 py-0">
                    <button
                      onClick={() => setExpandedId(isExpanded ? null : f.id)}
                      className="flex w-full cursor-pointer items-center text-left"
                    >
                      <span
                        className="w-1/5 px-4 py-3 text-text"
                        title={fmtAbsolute(f.referenceTime)}
                      >
                        {fmtDateTime(f.referenceTime)}
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
                        {formatBounds(f.bounds)}
                      </span>
                      <span className="w-1/5 px-4 py-3 text-text-secondary">
                        {f.files.length} ({formatBytes(totalSize)})
                      </span>
                      <span className="w-1/5 px-4 py-3 text-text-muted">
                        <TimeAgo iso={f.createdAt} />
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
                                    onCopied={copyToast}
                                  />
                                </td>
                                <td className="py-0.5 pr-4 text-text-secondary">
                                  {file.variable}
                                </td>
                                <td
                                  className="py-0.5 pr-4 text-text-muted"
                                  title={fmtAbsolute(file.validTime)}
                                >
                                  {fmtDateTime(file.validTime)}
                                </td>
                                <td
                                  className="py-0.5 pr-4 text-text-muted"
                                  title={fmtAbsolute(file.validUntilTime)}
                                >
                                  {fmtDateTime(file.validUntilTime)}
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
                  {isLoading ? "Loading..." : "No forecasts found."}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      <div className="space-y-3 md:hidden">
        {filtered.map((f) => {
          const totalSize = f.files.reduce(
            (sum, file) => sum + file.fileSize,
            0
          )
          const isExpanded = expandedId === f.id

          return (
            <AdminCard key={f.id}>
              <AdminCardHeader>
                <span
                  className="font-medium text-text"
                  title={fmtAbsolute(f.referenceTime)}
                >
                  {fmtDateTime(f.referenceTime)}
                </span>
                <CopyIdCell id={f.id} onCopied={copyToast} />
              </AdminCardHeader>
              <AdminCardField label="Attribution">
                <a
                  href={f.attribution.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:underline"
                >
                  {f.attribution.text}
                </a>
              </AdminCardField>
              <AdminCardField label="Bounds">
                {formatBounds(f.bounds)}
              </AdminCardField>
              <AdminCardField label="Files">
                {f.files.length} ({formatBytes(totalSize)})
              </AdminCardField>
              <AdminCardField label="Created">
                <TimeAgo iso={f.createdAt} />
              </AdminCardField>
              {f.files.length > 0 && (
                <AdminCardFooter>
                  <button
                    onClick={() => setExpandedId(isExpanded ? null : f.id)}
                    aria-expanded={isExpanded}
                    className="cursor-pointer text-sm text-text-secondary transition-colors hover:text-text"
                  >
                    {isExpanded
                      ? "Hide files"
                      : `Show files (${f.files.length})`}
                  </button>
                </AdminCardFooter>
              )}
              {isExpanded && f.files.length > 0 && (
                <div className="mt-3 space-y-3 border-t border-border pt-3">
                  {f.files.map((file) => (
                    <div
                      key={file.id}
                      className="rounded border border-border bg-surface p-2 text-xs"
                    >
                      <div className="mb-1 flex items-center justify-between gap-2">
                        <span className="font-medium text-text">
                          {file.variable}
                        </span>
                        <CopyIdCell id={file.id} onCopied={copyToast} />
                      </div>
                      <div
                        className="text-text-muted"
                        title={`Valid ${fmtAbsolute(file.validTime)} - ${fmtAbsolute(file.validUntilTime)}`}
                      >
                        Valid: {fmtDateTime(file.validTime)} -{" "}
                        {fmtDateTime(file.validUntilTime)}
                      </div>
                      <div className="text-text-muted">
                        Size: {formatBytes(file.fileSize)}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </AdminCard>
          )
        })}
        {filtered.length === 0 && (
          <Card className="px-4 py-6 text-center text-sm text-text-muted">
            {isLoading ? "Loading..." : "No forecasts found."}
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
