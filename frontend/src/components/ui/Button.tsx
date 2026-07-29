import { Link, type LinkProps } from "react-router"

type ButtonVariant = "primary" | "secondary" | "danger" | "ghost"

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

const baseClasses =
  "inline-flex min-h-11 cursor-pointer items-center justify-center rounded px-4 py-2 text-sm font-medium transition-colors"

interface CommonProps {
  variant?: ButtonVariant
  className?: string
}

type LinkButtonProps = CommonProps &
  Omit<LinkProps, keyof CommonProps> & {
    to: LinkProps["to"]
    href?: never
  }

type AnchorButtonProps = CommonProps &
  Omit<React.AnchorHTMLAttributes<HTMLAnchorElement>, keyof CommonProps> & {
    href: string
    to?: never
  }

type NativeButtonProps = CommonProps &
  Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, keyof CommonProps> & {
    to?: never
    href?: never
  }

type ButtonProps = LinkButtonProps | AnchorButtonProps | NativeButtonProps

/**
 * Themed button-styled control.
 *
 * Pass `to` to render a React Router `<Link>` (client-side navigation),
 * `href` to render a plain `<a>` (external links), or neither to render a
 * native `<button>`. Using a real anchor for navigation preserves
 * right-click, middle-click, and open-in-new-tab semantics.
 *
 * @param variant - Visual style: primary, secondary, danger, or ghost.
 */
export default function Button(props: ButtonProps) {
  const { variant = "secondary", className = "" } = props
  const merged = `${baseClasses} ${variantClasses[variant]} ${className}`

  if (props.to !== undefined) {
    const { variant: _v, className: _c, ...rest } = props
    void _v
    void _c
    return <Link className={merged} {...rest} />
  }
  if (props.href !== undefined) {
    const { variant: _v, className: _c, ...rest } = props
    void _v
    void _c
    return <a className={merged} {...rest} />
  }
  const { variant: _v, className: _c, ...rest } = props
  void _v
  void _c
  return <button className={merged} {...rest} />
}
