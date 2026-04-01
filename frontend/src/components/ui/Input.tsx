import { forwardRef } from "react"

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  /** Label text displayed above the input. */
  label?: string
  /** Error message displayed below the input. */
  error?: string
}

/**
 * Themed text input with optional label and error display.
 * @param label - Label text shown above the input.
 * @param error - Validation error message.
 */
const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { label, error, id, className = "", ...props },
  ref
) {
  const inputId =
    id || (label ? label.toLowerCase().replace(/\s+/g, "-") : undefined)

  return (
    <div>
      {label && (
        <label
          htmlFor={inputId}
          className="mb-1 block text-sm font-medium text-text-secondary"
        >
          {label}
        </label>
      )}
      <input
        ref={ref}
        id={inputId}
        aria-invalid={error ? "true" : undefined}
        aria-describedby={error && inputId ? `${inputId}-error` : undefined}
        className={`w-full rounded border border-border bg-panel px-3 py-2 text-sm text-text placeholder-text-muted transition-colors focus:border-primary focus:outline-none ${error ? "border-error" : ""} ${className}`}
        {...props}
      />
      {error && (
        <p
          id={inputId ? `${inputId}-error` : undefined}
          role="alert"
          className="mt-1 text-sm text-error"
        >
          {error}
        </p>
      )}
    </div>
  )
})

export default Input
