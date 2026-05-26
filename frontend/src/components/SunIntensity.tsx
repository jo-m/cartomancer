const SUN_SECTORS = 10
const SUN_PAD = 30
const SUN_MAX_R = 50
const SUN_SIZE = (SUN_MAX_R + SUN_PAD) * 2
const SUN_CX = SUN_SIZE / 2
const SUN_CY = SUN_SIZE / 2
const DISC_R = 20
const RAY_INNER_R = 24
const RAY_OUTER_MIN_R = 30
const RAY_OUTER_MAX_R = SUN_MAX_R
const HALF_RAY_DEG = 360 / SUN_SECTORS / 2 - 3

/** Interpolates linearly between two RGB color stops. */
function lerpRgb(
  t: number,
  a: [number, number, number],
  b: [number, number, number]
): string {
  const r = Math.round(a[0] + (b[0] - a[0]) * t)
  const g = Math.round(a[1] + (b[1] - a[1]) * t)
  const bb = Math.round(a[2] + (b[2] - a[2]) * t)
  return `rgb(${r}, ${g}, ${bb})`
}

/** Returns the ray color for sector index i, going from cool to hot. */
function rayColor(i: number): string {
  // Pale yellow -> amber -> deep red across the ten sectors.
  const t = i / (SUN_SECTORS - 1)
  if (t < 0.5) {
    return lerpRgb(t / 0.5, [253, 224, 71], [245, 158, 11])
  }
  return lerpRgb((t - 0.5) / 0.5, [245, 158, 11], [185, 28, 28])
}

interface Props {
  /** The sun intensity index in [1, 10]. */
  value: number
}

/**
 * SunIntensity renders a dimensionless 1..10 sun intensity index as a
 * sunburst gauge: ten rays around a central disc light up progressively
 * with the value, and the value is shown numerically in the centre.
 */
export default function SunIntensity({ value }: Props) {
  const clamped = Math.max(1, Math.min(10, value))
  const rounded = Math.round(clamped)

  // Build the rays. Each sector is a wedge; lit sectors grow outward and warm
  // up in colour, while unlit sectors remain short and muted.
  const rays = Array.from({ length: SUN_SECTORS }, (_, i) => {
    const lit = i < rounded
    const angleDeg = (i * 360) / SUN_SECTORS - 90
    const outerR = lit ? RAY_OUTER_MAX_R : RAY_OUTER_MIN_R
    const a1 = ((angleDeg - HALF_RAY_DEG) * Math.PI) / 180
    const a2 = ((angleDeg + HALF_RAY_DEG) * Math.PI) / 180
    const x1 = SUN_CX + RAY_INNER_R * Math.cos(a1)
    const y1 = SUN_CY + RAY_INNER_R * Math.sin(a1)
    const x2 = SUN_CX + outerR * Math.cos(a1)
    const y2 = SUN_CY + outerR * Math.sin(a1)
    const x3 = SUN_CX + outerR * Math.cos(a2)
    const y3 = SUN_CY + outerR * Math.sin(a2)
    const x4 = SUN_CX + RAY_INNER_R * Math.cos(a2)
    const y4 = SUN_CY + RAY_INNER_R * Math.sin(a2)
    const fill = lit ? rayColor(i) : "var(--color-border)"
    return (
      <path
        key={i}
        d={`M${x1},${y1} L${x2},${y2} L${x3},${y3} L${x4},${y4} Z`}
        fill={fill}
        fillOpacity={lit ? 0.85 : 0.4}
        stroke={fill}
        strokeWidth={1}
      />
    )
  })

  // The central disc colour follows the overall intensity.
  const discFill = rayColor(rounded - 1)

  return (
    <div
      className="flex items-center gap-3"
      role="img"
      aria-label={`Sun intensity ${rounded} out of 10`}
    >
      <p className="text-xs font-medium text-text-muted [writing-mode:vertical-lr] rotate-180">
        Sun intensity
      </p>
      <svg
        width={SUN_SIZE}
        height={SUN_SIZE}
        viewBox={`0 0 ${SUN_SIZE} ${SUN_SIZE}`}
      >
        {/* Outer guide ring. */}
        <circle
          cx={SUN_CX}
          cy={SUN_CY}
          r={SUN_MAX_R}
          fill="none"
          stroke="var(--color-border)"
          strokeWidth={0.5}
        />
        {rays}
        {/* Central disc. */}
        <circle
          cx={SUN_CX}
          cy={SUN_CY}
          r={DISC_R}
          fill={discFill}
          fillOpacity={0.9}
          stroke={discFill}
          strokeWidth={1}
        />
        <text
          x={SUN_CX}
          y={SUN_CY}
          textAnchor="middle"
          dominantBaseline="central"
          fontSize={18}
          fontWeight={700}
          fill="#ffffff"
        >
          {rounded}
        </text>
        <text
          x={SUN_CX}
          y={SUN_CY + DISC_R + 12}
          textAnchor="middle"
          dominantBaseline="central"
          fontSize={9}
          fill="var(--color-text-muted)"
        >
          / 10
        </text>
      </svg>
    </div>
  )
}
