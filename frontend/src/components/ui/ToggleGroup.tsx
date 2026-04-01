interface ToggleOption<T extends string> {
  value: T
  label: string
}

interface ToggleGroupProps<T extends string> {
  /** Available options in the group. */
  options: ToggleOption<T>[]
  /** Currently selected value. */
  value: T
  /** Called when the user selects a different option. */
  onChange: (value: T) => void
  /** Accessible label for the group. */
  ariaLabel?: string
  className?: string
}

/** Segmented toggle button group for mutually exclusive options. */
export default function ToggleGroup<T extends string>({
  options,
  value,
  onChange,
  ariaLabel,
  className = "",
}: ToggleGroupProps<T>) {
  return (
    <div
      role="radiogroup"
      aria-label={ariaLabel}
      className={`flex rounded border border-border text-sm ${className}`}
    >
      {options.map((opt) => (
        <button
          key={opt.value}
          type="button"
          role="radio"
          aria-checked={value === opt.value}
          onClick={() => onChange(opt.value)}
          className={`cursor-pointer px-3 py-1.5 first:rounded-l last:rounded-r transition-colors ${
            value === opt.value
              ? "bg-active text-active-text"
              : "text-text-secondary hover:bg-surface"
          }`}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}
