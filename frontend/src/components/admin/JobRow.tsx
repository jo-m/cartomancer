import { useState } from "react"
import CopyIdCell from "../CopyIdCell"
import TimeAgo from "../TimeAgo"
import { fmtAbsolute, fmtDateTime } from "../../lib/time"
import JobDetails from "./JobDetails"
import StatusBadge from "./StatusBadge"
import { rowDuration, type Job } from "./jobShared"

interface JobRowProps {
  job: Job
  onCopied: () => void
}

/** Single job row in the desktop table, with an expandable details panel. */
export default function JobRow({ job, onCopied }: JobRowProps) {
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
