import { useEffect, useState } from "react"

interface Props {
  /** The message to display in the toast. */
  message: string
  /** Called when the toast is dismissed (manually or after timeout). */
  onDismiss: () => void
}

const DURATION_MS = 3000
const FADE_MS = 500

/** Displays a transient error message that auto-dismisses after 3 seconds with a fade-out. */
export default function Toast({ message, onDismiss }: Props) {
  const [fading, setFading] = useState(false)

  useEffect(() => {
    const fadeTimer = setTimeout(() => setFading(true), DURATION_MS - FADE_MS)
    const dismissTimer = setTimeout(onDismiss, DURATION_MS)
    return () => {
      clearTimeout(fadeTimer)
      clearTimeout(dismissTimer)
    }
  }, [onDismiss])

  return (
    <div
      role="alert"
      className={`fixed bottom-4 left-1/2 z-50 -translate-x-1/2 flex items-center gap-3 rounded-lg bg-error px-4 py-3 text-sm text-primary-text shadow-lg transition-opacity duration-500 ${fading ? "opacity-0" : "opacity-100"}`}
    >
      <span>{message}</span>
      <button
        onClick={onDismiss}
        className="shrink-0 cursor-pointer text-primary-text/70 hover:text-primary-text transition-colors"
        aria-label="Dismiss notification"
      >
        Dismiss
      </button>
    </div>
  )
}
