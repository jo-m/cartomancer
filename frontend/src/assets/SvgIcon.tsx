interface SvgIconProps {
  svg: string
  className?: string
}

/** Renders a raw SVG string inline, inheriting color from the parent via currentColor. */
export default function SvgIcon({ svg, className }: SvgIconProps) {
  return (
    <span
      className={className}
      dangerouslySetInnerHTML={{ __html: svg }}
      style={{ display: "inline-flex" }}
      aria-hidden="true"
    />
  )
}
