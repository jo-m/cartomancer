const SUN_RAYS = 12
const SUN_MAX_R = 60
const SUN_PAD = 6
const SUN_SIZE = (SUN_MAX_R + SUN_PAD) * 2
const SUN_CX = SUN_SIZE / 2
const SUN_CY = SUN_SIZE / 2
const DISC_R = 22
const RAY_INNER_R = 25
const RAY_OUTER_LIT_R = SUN_MAX_R
const RAY_OUTER_UNLIT_R = 33
const RAY_BASE_HALF_W = 5

const SUN_INTENSITY_MIN = 0
const SUN_INTENSITY_MAX = SUN_RAYS

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

/**
 * Returns the warm fill color for a lit ray at sector index i, ramping from a
 * pale medieval gold through amber to deep crimson as i grows.
 */
function rayColor(i: number): string {
  const t = i / (SUN_RAYS - 1)
  if (t < 0.5) {
    return lerpRgb(t / 0.5, [240, 198, 110], [212, 140, 36])
  }
  return lerpRgb((t - 0.5) / 0.5, [212, 140, 36], [139, 32, 28])
}

/**
 * Returns the SVG path for a single curly tapered ray pointing along -Y in
 * the local frame, with the given outer radius. The ray has an asymmetric
 * S-curve so that all rays appear to swirl in the same rotational sense.
 */
function curlyRayPath(outerR: number): string {
  const w = RAY_BASE_HALF_W
  const innerR = RAY_INNER_R
  const len = outerR - innerR
  // Base sits at y = -innerR (just outside the central disc); tip is curled
  // slightly toward +X for a clockwise swirl.
  const tipX = 0.35 * w
  const tipY = -outerR
  const baseL = `${-w},${-innerR}`
  const baseR = `${w},${-innerR}`
  // Left edge: bulges outward near the base, then sweeps in toward the tip.
  const cp1 = `${-1.4 * w},${-(innerR + 0.3 * len)}`
  const cp2 = `${-0.15 * w},${-(innerR + 0.78 * len)}`
  // Right edge: curves back from the curled tip, then bulges outward.
  const cp3 = `${0.55 * w},${-(innerR + 0.85 * len)}`
  const cp4 = `${1.25 * w},${-(innerR + 0.4 * len)}`
  return [
    `M${baseL}`,
    `C${cp1} ${cp2} ${tipX},${tipY}`,
    `C${cp3} ${cp4} ${baseR}`,
    "Z",
  ].join(" ")
}

/**
 * Formats a broadband shortwave dose in J/m^2 for compact display. Uses MJ/m^2
 * for typical ride exposures, falling back to kJ/m^2 or J/m^2 for very small
 * values.
 */
function formatDose(doseJm2: number): string {
  if (doseJm2 >= 1e6) {
    return `${(doseJm2 / 1e6).toFixed(1)} MJ/m²`
  }
  if (doseJm2 >= 1e3) {
    return `${(doseJm2 / 1e3).toFixed(0)} kJ/m²`
  }
  return `${doseJm2.toFixed(0)} J/m²`
}

interface Props {
  /** The sun intensity index in [0, 12]. */
  value: number
  /** Integrated broadband shortwave dose along the track in J/m^2. */
  doseJm2?: number
}

/**
 * SunIntensity renders a dimensionless sun intensity index in [0, 12] as a
 * medieval sun-in-splendour gauge: twelve curly tapered rays around a central
 * disc light up progressively with the value, the value is shown numerically
 * in the centre, and the integrated broadband shortwave dose is displayed
 * below the gauge.
 */
export default function SunIntensity({ value, doseJm2 }: Props) {
  const clamped = Math.max(
    SUN_INTENSITY_MIN,
    Math.min(SUN_INTENSITY_MAX, value)
  )
  const rounded = Math.round(clamped)

  // Build twelve curly rays evenly spaced around the disc. Lit rays grow
  // outward and warm up in colour; unlit rays remain short stubs in a muted
  // border tone.
  const rays = Array.from({ length: SUN_RAYS }, (_, i) => {
    const lit = i < rounded
    const outerR = lit ? RAY_OUTER_LIT_R : RAY_OUTER_UNLIT_R
    const fill = lit ? rayColor(i) : "var(--color-border)"
    const stroke = lit ? rayColor(i) : "var(--color-border)"
    return (
      <g
        key={i}
        transform={`translate(${SUN_CX},${SUN_CY}) rotate(${(i * 360) / SUN_RAYS})`}
      >
        <path
          d={curlyRayPath(outerR)}
          fill={fill}
          fillOpacity={lit ? 0.92 : 0.45}
          stroke={stroke}
          strokeOpacity={lit ? 1 : 0.6}
          strokeWidth={0.6}
          strokeLinejoin="round"
        />
      </g>
    )
  })

  // The central disc colour follows the overall intensity. At zero the disc
  // takes the muted border colour to match the unlit rays.
  const discFill = rounded > 0 ? rayColor(rounded - 1) : "var(--color-border)"
  const discStroke =
    rounded > 0
      ? rayColor(Math.min(rounded, SUN_RAYS - 1))
      : "var(--color-border)"

  return (
    <div
      className="flex items-center gap-3"
      role="img"
      aria-label={`Sun intensity ${rounded} out of ${SUN_INTENSITY_MAX}`}
    >
      <p className="text-xs font-medium text-text-muted [writing-mode:vertical-lr] rotate-180">
        Sun intensity
      </p>
      <div className="flex flex-col items-center gap-1">
        <svg
          width={SUN_SIZE}
          height={SUN_SIZE}
          viewBox={`0 0 ${SUN_SIZE} ${SUN_SIZE}`}
        >
          {rays}
          <circle
            cx={SUN_CX}
            cy={SUN_CY}
            r={DISC_R}
            fill={discFill}
            fillOpacity={0.95}
            stroke={discStroke}
            strokeWidth={1.2}
          />
          <text
            x={SUN_CX}
            y={SUN_CY - 2}
            textAnchor="middle"
            dominantBaseline="central"
            fontSize={20}
            fontWeight={700}
            fill="#faf6ef"
          >
            {rounded}
          </text>
          <text
            x={SUN_CX}
            y={SUN_CY + 11}
            textAnchor="middle"
            dominantBaseline="central"
            fontSize={9}
            fill="#faf6ef"
            fillOpacity={0.85}
          >
            / {SUN_INTENSITY_MAX}
          </text>
        </svg>
        {doseJm2 != null && Number.isFinite(doseJm2) && (
          <p className="text-[10px] text-text-muted">
            Dose: {formatDose(doseJm2)}
          </p>
        )}
      </div>
    </div>
  )
}
