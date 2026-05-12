import { useMemo } from "react"
import type { ForecastPoint } from "../types/forecast"

const ROSE_SECTORS = 8
const ROSE_PAD = 30
const ROSE_MAX_R = 50
const ROSE_SIZE = (ROSE_MAX_R + ROSE_PAD) * 2
const ROSE_CX = ROSE_SIZE / 2
const ROSE_CY = ROSE_SIZE / 2

export interface WindRoseConfig {
  /** Direction field to bin by. */
  directionKey: "relativeWindDirectionDeg" | "windDirectionDeg"
  /** Labels for the 8 sectors (cardinal positions only). */
  labels: readonly string[]
  /** Returns the fill color for a given sector index. */
  sectorColor: (i: number) => string
  /** Vertical label displayed beside the rose. */
  title: string
}

/** Renders a wind rose SVG from forecast points using the given configuration. */
export default function WindRose({
  points,
  config,
}: {
  points: ForecastPoint[]
  config: WindRoseConfig
}) {
  const roseData = useMemo(() => {
    const bins = new Array(ROSE_SECTORS).fill(0)
    let totalWeight = 0
    for (const p of points) {
      const dir = p[config.directionKey]
      if (dir == null || p.windSpeedMs == null || p.windSpeedMs === 0) continue
      const sector = Math.round(dir / (360 / ROSE_SECTORS)) % ROSE_SECTORS
      bins[sector] += p.windSpeedMs
      totalWeight += p.windSpeedMs
    }
    if (totalWeight === 0) return null
    const maxBin = Math.max(...bins)
    return bins.map((v) => (maxBin > 0 ? v / maxBin : 0))
  }, [points, config.directionKey])

  if (!roseData) return null

  // Build petal paths as filled triangular wedges.
  const petals = roseData.map((fraction, i) => {
    if (fraction === 0) return null
    const angleDeg = (i * 360) / ROSE_SECTORS - 90
    const halfWedge = 360 / ROSE_SECTORS / 2 - 2
    const r = ROSE_MAX_R * fraction
    const a1 = ((angleDeg - halfWedge) * Math.PI) / 180
    const a2 = ((angleDeg + halfWedge) * Math.PI) / 180
    const x1 = ROSE_CX + r * Math.cos(a1)
    const y1 = ROSE_CY + r * Math.sin(a1)
    const x2 = ROSE_CX + r * Math.cos(a2)
    const y2 = ROSE_CY + r * Math.sin(a2)

    const fill = config.sectorColor(i)

    return (
      <path
        key={i}
        d={`M${ROSE_CX},${ROSE_CY} L${x1},${y1} A${r},${r} 0 0,1 ${x2},${y2} Z`}
        fill={fill}
        fillOpacity={0.5}
        stroke={fill}
        strokeWidth={1}
      />
    )
  })

  // Label positions at the cardinal slots.
  const labels = config.labels.map((label, i) => {
    if (!label) return null
    const angleDeg = (i * 360) / ROSE_SECTORS - 90
    const r = ROSE_MAX_R + 14
    const x = ROSE_CX + r * Math.cos((angleDeg * Math.PI) / 180)
    const y = ROSE_CY + r * Math.sin((angleDeg * Math.PI) / 180)
    return (
      <text
        key={i}
        x={x}
        y={y}
        textAnchor="middle"
        dominantBaseline="central"
        fontSize={10}
        fill="var(--color-text-muted)"
      >
        {label}
      </text>
    )
  })

  return (
    <div className="flex items-center gap-3">
      <p className="text-xs font-medium text-text-muted [writing-mode:vertical-lr] rotate-180">
        {config.title}
      </p>
      <svg
        width={ROSE_SIZE}
        height={ROSE_SIZE}
        viewBox={`0 0 ${ROSE_SIZE} ${ROSE_SIZE}`}
      >
        {/* Guide circles. */}
        <circle
          cx={ROSE_CX}
          cy={ROSE_CY}
          r={ROSE_MAX_R}
          fill="none"
          stroke="var(--color-border)"
          strokeWidth={0.5}
        />
        <circle
          cx={ROSE_CX}
          cy={ROSE_CY}
          r={ROSE_MAX_R / 2}
          fill="none"
          stroke="var(--color-border)"
          strokeWidth={0.5}
        />
        {/* Cross hairs. */}
        <line
          x1={ROSE_CX}
          y1={ROSE_CY - ROSE_MAX_R}
          x2={ROSE_CX}
          y2={ROSE_CY + ROSE_MAX_R}
          stroke="var(--color-border)"
          strokeWidth={0.5}
        />
        <line
          x1={ROSE_CX - ROSE_MAX_R}
          y1={ROSE_CY}
          x2={ROSE_CX + ROSE_MAX_R}
          y2={ROSE_CY}
          stroke="var(--color-border)"
          strokeWidth={0.5}
        />
        {petals}
        {labels}
      </svg>
    </div>
  )
}
