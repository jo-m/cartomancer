import { useCallback, useMemo, memo } from "react"
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
} from "recharts"
import { useHoverValue, type HoverStore } from "../hooks/useHoverSync"

interface TrackPoint {
  lat: number
  lon: number
  ele: number
  d: number
}

interface Props {
  points: TrackPoint[]
  hoverStore: HoverStore
  color: string
}

/** Renders an interactive elevation profile chart using recharts. */
export default memo(function ElevationProfile({
  points,
  hoverStore,
  color,
}: Props) {
  const hoverIndex = useHoverValue(hoverStore)

  const data = useMemo(
    () =>
      points.map((p) => ({
        dKm: Math.round((p.d / 1000) * 100) / 100,
        ele: Math.round(p.ele),
      })),
    [points]
  )

  const [minEle, maxEle] = useMemo(() => {
    const elevations = data.map((d) => d.ele).filter(isFinite)
    const lo = elevations.length
      ? Math.floor(Math.min(...elevations) / 50) * 50
      : 0
    const hi = elevations.length
      ? Math.ceil(Math.max(...elevations) / 50) * 50
      : 1000
    return [lo, hi]
  }, [data])

  const handleMouseMove = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (state: any) => {
      if (state?.activeTooltipIndex != null) {
        hoverStore.set(state.activeTooltipIndex as number)
      }
    },
    [hoverStore]
  )

  const handleMouseLeave = useCallback(() => {
    hoverStore.set(null)
  }, [hoverStore])

  const hoveredDKm = hoverIndex != null ? data[hoverIndex]?.dKm : null

  return (
    <div className="mt-4">
      <p className="mb-1 text-xs font-medium text-gray-500">
        Elevation profile (m)
      </p>
      <ResponsiveContainer width="100%" height={180}>
        <AreaChart
          data={data}
          onMouseMove={handleMouseMove}
          onMouseLeave={handleMouseLeave}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
          <XAxis
            dataKey="dKm"
            type="number"
            domain={["dataMin", "dataMax"]}
            tickFormatter={(v: number) => `${v}`}
            tick={{ fontSize: 11, fill: "#6b7280" }}
            stroke="#d1d5db"
            label={{
              value: "km",
              position: "insideBottomRight",
              offset: -5,
              style: { fontSize: 10, fill: "#9ca3af" },
            }}
          />
          <YAxis
            domain={[minEle, maxEle]}
            tick={{ fontSize: 11, fill: "#6b7280" }}
            stroke="#d1d5db"
            width={44}
          />
          <Tooltip
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            formatter={(value: any) => [`${value} m`, "Elevation"]}
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            labelFormatter={(v: any) => `${v} km`}
            contentStyle={{
              fontSize: 12,
              borderColor: "#e5e7eb",
              borderRadius: 4,
            }}
          />
          {hoveredDKm != null && (
            <ReferenceLine x={hoveredDKm} stroke="#9ca3af" strokeWidth={1} />
          )}
          <Area
            type="monotone"
            dataKey="ele"
            stroke={color}
            strokeWidth={1.5}
            fill={color}
            fillOpacity={0.1}
            dot={false}
            activeDot={{ r: 3, fill: color }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
})
