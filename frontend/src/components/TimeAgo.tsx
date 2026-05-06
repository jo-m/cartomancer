import { fmtAbsolute, fmtRelative } from "../lib/time"

export interface TimeAgoProps {
  /** ISO 8601 timestamp to render. */
  iso: string
  /**
   * Optional override for the title attribute. Defaults to the long absolute
   * form (date + time + seconds + timezone).
   */
  title?: string
  className?: string
}

/**
 * Renders an ISO timestamp as a short relative phrase ("3 min ago") inside an
 * HTML <time> element, with the absolute timestamp exposed via the title
 * attribute for hover. Use this for any "when did X happen" surface where the
 * recency matters more than the wall-clock value.
 */
export default function TimeAgo({ iso, title, className }: TimeAgoProps) {
  return (
    <time
      dateTime={iso}
      title={title ?? fmtAbsolute(iso)}
      className={className}
    >
      {fmtRelative(iso)}
    </time>
  )
}
