import { useEffect, useState } from "react"

interface Props {
  /** The message to display in the toast. */
  message: string
  /** Visual variant: "error" (default) or "success". */
  variant?: "error" | "success"
  /** Called when the toast is dismissed (manually or after timeout). */
  onDismiss: () => void
}

const DURATION_MS = 3000
const FADE_MS = 500

const variantClasses = {
  error: "bg-error text-primary-text",
  success: "bg-success text-primary-text",
}

/** Displays a transient message that auto-dismisses after 3 seconds with a fade-out. */
export default function Toast({
  message,
  variant = "error",
  onDismiss,
}: Props) {
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
      className={`fixed bottom-4 left-1/2 z-50 -translate-x-1/2 flex items-center gap-3 rounded-lg ${variantClasses[variant]} px-4 py-3 text-sm shadow-lg transition-opacity duration-500 ${fading ? "opacity-0" : "opacity-100"}`}
    >
      <span>{message}</span>
      <button
        onClick={onDismiss}
        className="inline-flex min-h-11 shrink-0 cursor-pointer items-center px-2 text-primary-text/70 hover:text-primary-text transition-colors"
        aria-label="Dismiss notification"
      >
        Dismiss
      </button>
    </div>
  )
}
