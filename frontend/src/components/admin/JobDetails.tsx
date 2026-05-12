import SectionHeading from "../ui/SectionHeading"
import type { Job } from "./jobShared"

interface JobDetailsProps {
  job: Job
}

/** Expandable details panel: error message, raw args JSON, retry config. */
export default function JobDetails({ job }: JobDetailsProps) {
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
