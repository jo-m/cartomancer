import { useCallback, useState } from "react"

interface ToastState {
  key: number
  message: string
  variant: "error" | "success"
}

/**
 * Manages toast notification state with support for re-triggering identical messages.
 *
 * @returns toast - current toast state (null when dismissed), showToast - display a message, dismissToast - clear the toast
 */
export default function useToast() {
  const [toast, setToast] = useState<ToastState | null>(null)

  const showToast = useCallback(
    (message: string, variant: "error" | "success" = "error") => {
      setToast((prev) => ({
        key: (prev?.key ?? 0) + 1,
        message,
        variant,
      }))
    },
    []
  )

  const dismissToast = useCallback(() => setToast(null), [])

  return { toast, showToast, dismissToast } as const
}
