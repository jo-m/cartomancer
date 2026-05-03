import { forwardRef } from "react"

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  /** Label text displayed above the select. */
  label?: string
}

/**
 * Themed select dropdown with optional label.
 * @param label - Label text shown above the select.
 */
const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { label, id, className = "", children, ...props },
  ref
) {
  const selectId =
    id || (label ? label.toLowerCase().replace(/\s+/g, "-") : undefined)

  return (
    <div>
      {label && (
        <label
          htmlFor={selectId}
          className="mb-1 block text-sm font-medium text-text-secondary"
        >
          {label}
        </label>
      )}
      <select
        ref={ref}
        id={selectId}
        className={`min-h-11 cursor-pointer rounded border border-border bg-panel px-3 py-2 text-sm text-text transition-colors focus:border-primary focus:outline-none ${className}`}
        {...props}
      >
        {children}
      </select>
    </div>
  )
})

export default Select
