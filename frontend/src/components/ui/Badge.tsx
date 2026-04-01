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
          className="cursor-pointer leading-none text-text-muted hover:text-text-secondary"
          aria-label="Remove"
        >
          &times;
        </button>
      )}
    </span>
  )
}
