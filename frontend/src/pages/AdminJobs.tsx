import { useMemo, useState } from "react"
import { $api } from "../api/client"
import useDocumentTitle from "../hooks/useDocumentTitle"
import useToast from "../hooks/useToast"
import {
  boolParam,
  enumParam,
  numberParam,
  stringParam,
  useUrlState,
} from "../hooks/useUrlState"
import PageContainer from "../components/ui/PageContainer"
import Card from "../components/ui/Card"
import Input from "../components/ui/Input"
import Select from "../components/ui/Select"
import Button from "../components/ui/Button"
import SectionHeading from "../components/ui/SectionHeading"
import Toast from "../components/Toast"
import AdminTabs from "../components/admin/AdminTabs"
import JobRow from "../components/admin/JobRow"
import JobMobileCard from "../components/admin/JobMobileCard"
import StatusBadge from "../components/admin/StatusBadge"
import {
  STATUS_FILTERS,
  STATUS_LABEL,
  type StatusCode,
  type StatusFilter,
} from "../components/admin/jobShared"

const AUTO_REFRESH_MS = 5000

export default function AdminJobs() {
  useDocumentTitle("Jobs")
  const { toast, showToast, dismissToast } = useToast()

  const searchSchema = useMemo(
    () => ({
      kind: stringParam(),
      status: enumParam<StatusFilter>("", STATUS_FILTERS),
      errorOnly: boolParam(),
      limit: numberParam(200),
    }),
    []
  )
  const [urlState, setUrlState] = useUrlState(searchSchema)
  const [paused, setPaused] = useState(false)

  const queryParams: Record<string, string | boolean | number> = {}
  if (urlState.kind) queryParams.kind = urlState.kind
  if (urlState.status) queryParams.status = urlState.status
  if (urlState.errorOnly) queryParams.errorOnly = true
  if (urlState.limit !== 200) queryParams.limit = urlState.limit

  const { data, isLoading, isFetching, refetch, error } = $api.useQuery(
    "get",
    "/admin/jobs",
    { params: { query: queryParams } },
    { refetchInterval: paused ? false : AUTO_REFRESH_MS }
  )

  const runnerIds = data?.runnerIds ?? []
  const statusCounts = data?.statusCounts ?? []
  const byKind = data?.byKind ?? []
  const jobs = data?.jobs ?? []
  const truncated = data?.truncated ?? false

  const totalCount = statusCounts.reduce((sum, r) => sum + r.count, 0)
  const countByStatus = new Map<string, number>()
  for (const row of statusCounts) countByStatus.set(row.status, row.count)

  function copyToast() {
    showToast("Copied to clipboard", "success")
  }

  return (
    <PageContainer size="2xl">
      <AdminTabs current="jobs" />

      <Card className="mb-4 p-4">
        <div className="flex flex-wrap items-start gap-x-6 gap-y-3">
          <div>
            <SectionHeading>Workers</SectionHeading>
            <div
              className="mt-1 text-sm"
              title={runnerIds.join("\n") || undefined}
            >
              {runnerIds.length === 0 ? (
                <span className="text-error" role="alert">
                  No worker registered
                </span>
              ) : runnerIds.length === 1 ? (
                <span className="text-success">1 registered</span>
              ) : (
                <span className="text-error" role="alert">
                  {runnerIds.length} registered (split brain)
                </span>
              )}
            </div>
          </div>

          <div>
            <SectionHeading>Total rows</SectionHeading>
            <div className="mt-1 text-sm text-text">{totalCount}</div>
          </div>

          {(["C", "R", "A", "E", "S"] as StatusCode[]).map((code) => (
            <div key={code}>
              <SectionHeading>{STATUS_LABEL[code]}</SectionHeading>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-lg font-semibold text-text">
                  {countByStatus.get(code) ?? 0}
                </span>
                <StatusBadge status={code} />
              </div>
            </div>
          ))}

          <div className="ml-auto flex items-center gap-2">
            <Button
              variant={paused ? "secondary" : "ghost"}
              onClick={() => setPaused((p) => !p)}
              aria-pressed={paused}
            >
              {paused ? "Resume" : "Pause"} auto-refresh
            </Button>
            <Button
              variant="ghost"
              onClick={() => void refetch()}
              disabled={isFetching}
            >
              Refresh
            </Button>
          </div>
        </div>
        <p className="mt-3 text-xs text-text-muted">
          Jobs are auto-cleaned shortly after they succeed or exhaust their
          retries, so this view always reflects the current queue, not full
          history.
        </p>
        {error && (
          <p className="mt-2 text-xs text-error" role="alert">
            Failed to load jobs: {error.message}
          </p>
        )}
      </Card>

      {byKind.length > 0 && (
        <Card className="mb-4 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-border text-xs font-medium text-text-muted">
                <th className="px-4 py-2">Kind</th>
                <th className="px-4 py-2">Total</th>
                <th className="px-4 py-2">Created</th>
                <th className="px-4 py-2">Running</th>
                <th className="px-4 py-2">Errored</th>
                <th className="px-4 py-2">Aborted</th>
                <th className="px-4 py-2">Succeeded</th>
                <th className="px-4 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {byKind.map((row) => (
                <tr
                  key={row.kind}
                  className="border-b border-border last:border-0"
                >
                  <td className="px-4 py-2 font-mono text-xs text-text">
                    {row.kind}
                  </td>
                  <td className="px-4 py-2 text-text-secondary">{row.total}</td>
                  <td className="px-4 py-2 text-text-secondary">
                    {row.created}
                  </td>
                  <td className="px-4 py-2 text-text-secondary">
                    {row.running}
                  </td>
                  <td className="px-4 py-2 text-text-secondary">
                    {row.errored}
                  </td>
                  <td className="px-4 py-2 text-text-secondary">
                    {row.aborted}
                  </td>
                  <td className="px-4 py-2 text-text-secondary">
                    {row.succeeded}
                  </td>
                  <td className="px-4 py-2 text-right">
                    <button
                      type="button"
                      onClick={() => setUrlState({ kind: row.kind })}
                      className="cursor-pointer text-xs text-text-muted hover:text-text transition-colors"
                    >
                      Filter
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}

      <div className="mb-4 flex flex-wrap items-end gap-3">
        <div className="flex-1 min-w-48">
          <Input
            type="text"
            placeholder="Filter by kind..."
            value={urlState.kind}
            onChange={(e) => setUrlState({ kind: e.target.value })}
            aria-label="Filter by kind"
          />
        </div>
        <Select
          aria-label="Status filter"
          value={urlState.status}
          onChange={(e) =>
            setUrlState({ status: e.target.value as StatusFilter })
          }
        >
          <option value="">All statuses</option>
          {(["C", "R", "A", "E", "S"] as StatusCode[]).map((code) => (
            <option key={code} value={code}>
              {STATUS_LABEL[code]}
            </option>
          ))}
        </Select>
        <label className="flex cursor-pointer items-center gap-2 text-sm text-text-secondary">
          <input
            type="checkbox"
            checked={urlState.errorOnly}
            onChange={(e) => setUrlState({ errorOnly: e.target.checked })}
            className="cursor-pointer"
          />
          Only with error
        </label>
        <Select
          aria-label="Page size"
          value={String(urlState.limit)}
          onChange={(e) =>
            setUrlState({ limit: Number(e.target.value) || 200 })
          }
        >
          <option value="50">50 rows</option>
          <option value="200">200 rows</option>
          <option value="500">500 rows</option>
          <option value="1000">1000 rows</option>
        </Select>
      </div>

      <Card className="hidden overflow-x-auto md:block">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs font-medium text-text-muted">
              <th className="px-3 py-2">ID</th>
              <th className="px-3 py-2">Kind</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2">Attempts</th>
              <th className="px-3 py-2">Created</th>
              <th className="px-3 py-2">Started</th>
              <th className="px-3 py-2">Finished</th>
              <th className="px-3 py-2">Duration</th>
              <th className="px-3 py-2">Next attempt</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((j) => (
              <JobRow key={j.id} job={j} onCopied={copyToast} />
            ))}
            {jobs.length === 0 && (
              <tr>
                <td
                  colSpan={9}
                  className="px-4 py-6 text-center text-sm text-text-muted"
                >
                  {isLoading ? "Loading..." : "No jobs found."}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      <div className="space-y-3 md:hidden">
        {jobs.map((j) => (
          <JobMobileCard key={j.id} job={j} onCopied={copyToast} />
        ))}
        {jobs.length === 0 && (
          <Card className="px-4 py-6 text-center text-sm text-text-muted">
            {isLoading ? "Loading..." : "No jobs found."}
          </Card>
        )}
      </div>

      {truncated && (
        <p className="mt-3 text-xs text-text-muted">
          Result was truncated at {urlState.limit} rows. Increase the page size
          or refine the filters to see more.
        </p>
      )}

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
