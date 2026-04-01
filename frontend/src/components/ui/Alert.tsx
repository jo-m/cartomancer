type AlertVariant = "info" | "warning" | "error" | "success"

interface AlertProps {
  variant?: AlertVariant
  children: React.ReactNode
  className?: string
}

const variantClasses: Record<AlertVariant, string> = {
  info: "border-info-border bg-info-light text-info",
  warning: "border-warning-border bg-warning-light text-warning-text",
  error: "border-error-border bg-error-light text-error",
  success: "border-border bg-success-light text-success",
}

/** Themed alert box for info, warning, error, and success messages. */
export default function Alert({
  variant = "info",
  children,
  className = "",
}: AlertProps) {
  return (
    <div
      role="alert"
      className={`rounded border px-3 py-2 text-sm ${variantClasses[variant]} ${className}`}
    >
      {children}
    </div>
  )
}
