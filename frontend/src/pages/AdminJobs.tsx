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
import CopyIdCell from "../components/CopyIdCell"
import AdminTabs from "../components/admin/AdminTabs"
import AdminCard, {
  AdminCardField,
  AdminCardFooter,
  AdminCardHeader,
} from "../components/admin/AdminCard"
import TimeAgo from "../components/TimeAgo"
import { fmtAbsolute, fmtDateTime } from "../lib/time"

type StatusCode = "C" | "R" | "A" | "E" | "S"
type StatusFilter = "" | StatusCode

const STATUS_FILTERS = ["", "C", "R", "A", "E", "S"] as const

const STATUS_LABEL: Record<StatusCode, string> = {
  C: "Created",
  R: "Running",
  A: "Aborted",
  E: "Errored",
  S: "Succeeded",
}

/**
 * Maps a status code to a Tailwind class set for its colored badge.
 * Tokens come from the project theme (`error`, `success`, etc.).
 */
function statusBadgeClass(code: string): string {
  switch (code) {
    case "C":
      return "border-border bg-surface text-text-secondary"
    case "R":
      return "border-primary/40 bg-primary/10 text-primary"
    case "S":
      return "border-success/40 bg-success/10 text-success"
    case "E":
      return "border-error/40 bg-error/10 text-error"
    case "A":
      return "border-star/40 bg-star/10 text-star"
    default:
      return "border-border bg-surface text-text-muted"
  }
}

interface StatusBadgeProps {
  status: string
}

/** Compact colored pill that renders the human-readable status name. */
function StatusBadge({ status }: StatusBadgeProps) {
  const label = (STATUS_LABEL as Record<string, string>)[status] ?? status
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium ${statusBadgeClass(status)}`}
    >
      {status === "R" && (
        <span
          className="h-1.5 w-1.5 animate-pulse rounded-full bg-primary"
          aria-hidden="true"
        />
      )}
      {label}
    </span>
  )
}

/**
 * Formats a millisecond duration as a human-readable string.
 * Sub-second durations are shown in ms, otherwise s / m / h.
 */
function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return "--"
  if (ms < 1000) return `${Math.round(ms)}ms`
  const s = ms / 1000
  if (s < 60) return `${s.toFixed(s < 10 ? 1 : 0)}s`
  const m = Math.floor(s / 60)
  const remS = Math.round(s - m * 60)
  if (m < 60) return `${m}m ${remS}s`
  const h = Math.floor(m / 60)
  const remM = m - h * 60
  return `${h}h ${remM}m`
}

interface RowDuration {
  text: string
  running: boolean
}

/**
 * Computes the duration text shown for a single job row. Returns the elapsed
 * runtime for a finished job, or the live-elapsed time since `startedAt` for
 * a still-running job. Returns null when no start time is available.
 */
function rowDuration(job: {
  startedAt?: string | null
  finishedAt?: string | null
  status: string
}): RowDuration | null {
  if (!job.startedAt) return null
  const start = new Date(job.startedAt).getTime()
  if (job.finishedAt) {
    const end = new Date(job.finishedAt).getTime()
    return { text: formatDuration(end - start), running: false }
  }
  if (job.status === "R") {
    return { text: formatDuration(Date.now() - start), running: true }
  }
  return null
}

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

interface JobRowProps {
  job: {
    id: number
    kind: string
    status: string
    createdAt: string
    startedAt?: string
    finishedAt?: string
    nextAttemptAt?: string
    /** Random worker process id; arrives as a string to preserve precision. */
    runnerId?: string
    attempts: number
    maxAttempts: number
    delayS: number
    backoffFactorS: number
    argsJson: string
    error?: string
  }
  onCopied: () => void
}

/** Single job row in the desktop table, with an expandable details panel. */
function JobRow({ job, onCopied }: JobRowProps) {
  const [expanded, setExpanded] = useState(false)
  const dur = rowDuration(job)
  const hasDetails = Boolean(job.error || job.argsJson)

  return (
    <>
      <tr
        className="border-b border-border last:border-0 cursor-pointer hover:bg-surface transition-colors"
        onClick={() => hasDetails && setExpanded((v) => !v)}
        aria-expanded={hasDetails ? expanded : undefined}
      >
        <td className="px-3 py-2 align-top">
          <CopyIdCell id={job.id} onCopied={onCopied} />
        </td>
        <td className="px-3 py-2 align-top font-mono text-xs text-text">
          {job.kind}
        </td>
        <td className="px-3 py-2 align-top">
          <StatusBadge status={job.status} />
        </td>
        <td className="px-3 py-2 align-top text-text-secondary">
          {job.attempts}/{job.maxAttempts}
        </td>
        <td
          className="px-3 py-2 align-top text-text-muted"
          title={fmtAbsolute(job.createdAt)}
        >
          <TimeAgo iso={job.createdAt} />
        </td>
        <td
          className="px-3 py-2 align-top text-text-muted"
          title={job.startedAt ? fmtAbsolute(job.startedAt) : undefined}
        >
          {job.startedAt ? <TimeAgo iso={job.startedAt} /> : "--"}
        </td>
        <td
          className="px-3 py-2 align-top text-text-muted"
          title={job.finishedAt ? fmtAbsolute(job.finishedAt) : undefined}
        >
          {job.finishedAt ? <TimeAgo iso={job.finishedAt} /> : "--"}
        </td>
        <td className="px-3 py-2 align-top text-text-secondary">
          {dur ? (
            <span className={dur.running ? "text-primary" : undefined}>
              {dur.text}
              {dur.running ? "..." : ""}
            </span>
          ) : (
            "--"
          )}
        </td>
        <td
          className="px-3 py-2 align-top text-text-muted"
          title={job.nextAttemptAt ? fmtAbsolute(job.nextAttemptAt) : undefined}
        >
          {job.nextAttemptAt ? fmtDateTime(job.nextAttemptAt) : "--"}
        </td>
      </tr>
      {expanded && hasDetails && (
        <tr className="border-b border-border bg-surface">
          <td colSpan={9} className="px-3 py-3">
            <JobDetails job={job} />
          </td>
        </tr>
      )}
    </>
  )
}

interface JobMobileCardProps {
  job: JobRowProps["job"]
  onCopied: () => void
}

/** Mobile card view of a single job row. */
function JobMobileCard({ job, onCopied }: JobMobileCardProps) {
  const [expanded, setExpanded] = useState(false)
  const dur = rowDuration(job)
  const hasDetails = Boolean(job.error || job.argsJson)
  return (
    <AdminCard>
      <AdminCardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-mono text-xs text-text">{job.kind}</span>
          <StatusBadge status={job.status} />
        </div>
        <CopyIdCell id={job.id} onCopied={onCopied} />
      </AdminCardHeader>
      <AdminCardField label="Attempts">
        {job.attempts}/{job.maxAttempts}
      </AdminCardField>
      <AdminCardField label="Created">
        <TimeAgo iso={job.createdAt} />
      </AdminCardField>
      {job.startedAt && (
        <AdminCardField label="Started">
          <TimeAgo iso={job.startedAt} />
        </AdminCardField>
      )}
      {job.finishedAt && (
        <AdminCardField label="Finished">
          <TimeAgo iso={job.finishedAt} />
        </AdminCardField>
      )}
      {dur && (
        <AdminCardField label="Duration">
          <span className={dur.running ? "text-primary" : undefined}>
            {dur.text}
            {dur.running ? "..." : ""}
          </span>
        </AdminCardField>
      )}
      {job.nextAttemptAt && (
        <AdminCardField label="Next attempt">
          {fmtDateTime(job.nextAttemptAt)}
        </AdminCardField>
      )}
      {hasDetails && (
        <AdminCardFooter>
          <button
            type="button"
            onClick={() => setExpanded((v) => !v)}
            aria-expanded={expanded}
            className="cursor-pointer text-sm text-text-secondary transition-colors hover:text-text"
          >
            {expanded ? "Hide details" : "Show details"}
          </button>
        </AdminCardFooter>
      )}
      {expanded && hasDetails && (
        <div className="mt-3 border-t border-border pt-3">
          <JobDetails job={job} />
        </div>
      )}
    </AdminCard>
  )
}

interface JobDetailsProps {
  job: JobRowProps["job"]
}

/** Expandable details panel: error message, raw args JSON, retry config. */
function JobDetails({ job }: JobDetailsProps) {
  let prettyArgs = job.argsJson
  try {
    prettyArgs = JSON.stringify(JSON.parse(job.argsJson), null, 2)
  } catch {
    // Already a non-JSON string (shouldn't happen for valid rows).
  }
  return (
    <div className="space-y-3 text-xs">
      {job.error && (
        <div>
          <SectionHeading>Error</SectionHeading>
          <pre
            role="alert"
            className="mt-1 max-h-64 overflow-auto whitespace-pre-wrap rounded border border-error/40 bg-error/5 p-2 font-mono text-error"
          >
            {job.error}
          </pre>
        </div>
      )}
      <div>
        <SectionHeading>Args</SectionHeading>
        <pre className="mt-1 max-h-80 overflow-auto whitespace-pre-wrap rounded border border-border bg-panel p-2 font-mono text-text-secondary">
          {prettyArgs}
        </pre>
      </div>
      <div className="text-text-muted">
        delay={job.delayS}s, backoff factor={job.backoffFactorS}s
        {job.runnerId && (
          <>
            {", runner="}
            <span className="font-mono">{job.runnerId}</span>
          </>
        )}
      </div>
    </div>
  )
}
