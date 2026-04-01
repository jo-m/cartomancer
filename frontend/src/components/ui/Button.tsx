import { forwardRef } from "react"

type ButtonVariant = "primary" | "secondary" | "danger" | "ghost"

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
}

const variantClasses: Record<ButtonVariant, string> = {
  primary:
    "bg-primary text-primary-text hover:bg-primary-hover disabled:opacity-50 disabled:cursor-not-allowed",
  secondary:
    "border border-border bg-panel text-text-secondary hover:border-border-hover hover:bg-surface disabled:opacity-50 disabled:cursor-not-allowed",
  danger:
    "border border-error-border bg-panel text-error hover:bg-error-light disabled:opacity-50 disabled:cursor-not-allowed",
  ghost:
    "text-text-secondary hover:text-text disabled:opacity-50 disabled:cursor-not-allowed",
}

/**
 * Themed button with variant support.
 * @param variant - Visual style: primary, secondary, danger, or ghost.
 */
const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = "secondary", className = "", children, ...props },
  ref
) {
  return (
    <button
      ref={ref}
      className={`cursor-pointer rounded px-4 py-2 text-sm font-medium transition-colors ${variantClasses[variant]} ${className}`}
      {...props}
    >
      {children}
    </button>
  )
})

export default Button
