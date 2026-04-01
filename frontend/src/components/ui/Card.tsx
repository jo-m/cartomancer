interface CardProps {
  children: React.ReactNode
  className?: string
}

/** Themed card container with border and background. */
export default function Card({ children, className = "" }: CardProps) {
  return (
    <div className={`rounded-lg border border-border bg-panel ${className}`}>
      {children}
    </div>
  )
}
