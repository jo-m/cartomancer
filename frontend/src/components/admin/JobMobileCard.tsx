import { useState } from "react"
import CopyIdCell from "../CopyIdCell"
import TimeAgo from "../TimeAgo"
import { fmtDateTime } from "../../lib/time"
import AdminCard, {
  AdminCardField,
  AdminCardFooter,
  AdminCardHeader,
} from "./AdminCard"
import JobDetails from "./JobDetails"
import StatusBadge from "./StatusBadge"
import { rowDuration, type Job } from "./jobShared"

interface JobMobileCardProps {
  job: Job
  onCopied: () => void
}

/** Mobile card view of a single job row. */
export default function JobMobileCard({ job, onCopied }: JobMobileCardProps) {
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
