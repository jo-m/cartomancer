export type StatusCode = "C" | "R" | "A" | "E" | "S"
export type StatusFilter = "" | StatusCode

export const STATUS_FILTERS = ["", "C", "R", "A", "E", "S"] as const

export const STATUS_LABEL: Record<StatusCode, string> = {
  C: "Created",
  R: "Running",
  A: "Aborted",
  E: "Errored",
  S: "Succeeded",
}

export interface Job {
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

/**
 * Maps a status code to a Tailwind class set for its colored badge.
 * Tokens come from the project theme (`error`, `success`, etc.).
 */
export function statusBadgeClass(code: string): string {
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

export interface RowDuration {
  text: string
  running: boolean
}

/**
 * Computes the duration text shown for a single job row. Returns the elapsed
 * runtime for a finished job, or the live-elapsed time since `startedAt` for
 * a still-running job. Returns null when no start time is available.
 */
export function rowDuration(job: {
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
