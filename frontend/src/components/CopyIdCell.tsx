interface CopyIdCellProps {
  /** The id or uuid to copy when clicked. */
  id: string | number
  /** If set, only the first N characters of the id are shown. */
  truncate?: number
  /** Called after the value has been written to the clipboard. */
  onCopied?: () => void
}

/** Renders a monospaced id that copies its full value to the clipboard on click. */
export default function CopyIdCell({
  id,
  truncate,
  onCopied,
}: CopyIdCellProps) {
  const value = String(id)
  const display = truncate ? value.slice(0, truncate) : value

  async function handleClick(e: React.MouseEvent) {
    e.stopPropagation()
    e.preventDefault()
    try {
      await navigator.clipboard.writeText(value)
      onCopied?.()
    } catch {
      // Clipboard API may be unavailable (e.g. insecure context); silently ignore.
    }
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      title={`Click to copy: ${value}`}
      aria-label={`Copy id ${value}`}
      className="cursor-pointer font-mono text-xs text-text-muted hover:text-text transition-colors"
    >
      {display}
    </button>
  )
}
