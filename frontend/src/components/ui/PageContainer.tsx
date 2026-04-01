interface PageContainerProps {
  children: React.ReactNode
  /** Maximum width variant. */
  size?: "sm" | "md" | "lg" | "xl"
  className?: string
}

const sizeClasses = {
  sm: "max-w-lg",
  md: "max-w-2xl",
  lg: "max-w-3xl",
  xl: "max-w-5xl",
}

/** Consistent page wrapper with max-width and padding. */
export default function PageContainer({
  children,
  size = "xl",
  className = "",
}: PageContainerProps) {
  return (
    <div className={`mx-auto ${sizeClasses[size]} px-4 py-8 ${className}`}>
      {children}
    </div>
  )
}
