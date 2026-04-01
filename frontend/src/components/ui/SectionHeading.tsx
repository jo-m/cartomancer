interface SectionHeadingProps {
  children: React.ReactNode
  className?: string
}

/** Uppercase tracking-wide section label. */
export default function SectionHeading({
  children,
  className = "",
}: SectionHeadingProps) {
  return (
    <p
      className={`text-xs font-medium uppercase tracking-wide text-text-muted ${className}`}
    >
      {children}
    </p>
  )
}
