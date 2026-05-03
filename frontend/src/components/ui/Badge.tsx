interface BadgeProps {
  children: React.ReactNode
  className?: string
  /** Called when the remove button is clicked. If omitted, no remove button is shown. */
  onRemove?: () => void
}

/** Themed pill badge for tags, labels, and chips. */
export default function Badge({
  children,
  className = "",
  onRemove,
}: BadgeProps) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border border-tag-border bg-tag-bg px-2.5 py-0.5 text-xs text-tag-text ${className}`}
    >
      {children}
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          className="-mr-1 inline-flex h-6 w-6 cursor-pointer items-center justify-center leading-none text-text-muted hover:text-text-secondary"
          aria-label="Remove"
        >
          &times;
        </button>
      )}
    </span>
  )
}
