import { STATUS_LABEL, statusBadgeClass } from "./jobShared"

interface StatusBadgeProps {
  status: string
}

/** Compact colored pill that renders the human-readable status name. */
export default function StatusBadge({ status }: StatusBadgeProps) {
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
