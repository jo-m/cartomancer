interface AdminCardProps {
  children: React.ReactNode
  className?: string
}

/**
 * Mobile-friendly card replacing a single table row in admin tables.
 * Use together with AdminCardHeader, AdminCardField, and AdminCardFooter.
 */
export default function AdminCard({
  children,
  className = "",
}: AdminCardProps) {
  return (
    <div
      className={`rounded-lg border border-border bg-panel p-4 ${className}`}
    >
      {children}
    </div>
  )
}

interface AdminCardHeaderProps {
  children: React.ReactNode
}

/** Top section of an AdminCard, separated from the fields by a thin border. */
export function AdminCardHeader({ children }: AdminCardHeaderProps) {
  return (
    <div className="-mt-1 mb-2 flex items-start justify-between gap-2 border-b border-border pb-2">
      {children}
    </div>
  )
}

interface AdminCardFieldProps {
  /** Label shown on the left side of the row. */
  label: string
  children: React.ReactNode
}

/** Single label/value row inside an AdminCard. */
export function AdminCardField({ label, children }: AdminCardFieldProps) {
  return (
    <div className="flex items-baseline justify-between gap-3 py-0.5 text-sm">
      <span className="shrink-0 text-xs font-medium uppercase tracking-wide text-text-muted">
        {label}
      </span>
      <span className="min-w-0 break-words text-right text-text-secondary">
        {children}
      </span>
    </div>
  )
}

interface AdminCardFooterProps {
  children: React.ReactNode
}

/** Bottom section of an AdminCard, typically holding action buttons. */
export function AdminCardFooter({ children }: AdminCardFooterProps) {
  return (
    <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-border pt-3">
      {children}
    </div>
  )
}
