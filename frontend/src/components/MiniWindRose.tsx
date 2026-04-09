/** MiniWindRose renders a tiny 4-sector wind rose as an inline SVG. */
export default function MiniWindRose({
  head,
  right,
  tail,
  left,
}: {
  head?: number
  right?: number
  tail?: number
  left?: number
}) {
  const vals = [head ?? 0, right ?? 0, tail ?? 0, left ?? 0]
  const maxVal = Math.max(...vals)
  if (maxVal < 0.1) return null
  const fracs = vals.map((v) => v / maxVal)

  const colors = ["#ef4444", "#9ca3af", "#10b981", "#9ca3af"]
  const angles = [-90, 0, 90, 180]
  const cx = 16
  const cy = 16
  const maxR = 14
  const wedge = 35

  return (
    <svg
      width="32"
      height="32"
      viewBox="0 0 32 32"
      className="inline-block shrink-0"
      aria-label="Wind rose"
    >
      {fracs.map((f, i) => {
        if (f < 0.05) return null
        const r = maxR * f
        const a = angles[i]
        const a1 = ((a - wedge) * Math.PI) / 180
        const a2 = ((a + wedge) * Math.PI) / 180
        const x1 = cx + r * Math.cos(a1)
        const y1 = cy + r * Math.sin(a1)
        const x2 = cx + r * Math.cos(a2)
        const y2 = cy + r * Math.sin(a2)
        return (
          <path
            key={i}
            d={`M${cx},${cy} L${x1},${y1} A${r},${r} 0 0,1 ${x2},${y2} Z`}
            fill={colors[i]}
            fillOpacity={0.6}
            stroke={colors[i]}
            strokeWidth={0.5}
          />
        )
      })}
      <circle cx={cx} cy={cy} r={1} fill="var(--color-text-muted)" />
    </svg>
  )
}
