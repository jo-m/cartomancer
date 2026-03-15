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
import { fmtElapsed, fmtClock } from "../lib/time"

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
  /** Interpolated timestamps per track point index, if forecast is loaded. */
  forecastTimes?: number[]
}

interface ElevDatum {
  dKm: number
  ele: number
  ts: number | null
}

/** Renders an interactive elevation profile chart using recharts. */
export default memo(function ElevationProfile({
  points,
  hoverStore,
  color,
  forecastTimes,
}: Props) {
  const hoverIndex = useHoverValue(hoverStore)

  const data: ElevDatum[] = useMemo(
    () =>
      points.map((p, i) => ({
        dKm: Math.round((p.d / 1000) * 100) / 100,
        ele: Math.round(p.ele),
        ts: forecastTimes?.[i] ?? null,
      })),
    [points, forecastTimes]
  )

  const startTs = useMemo(() => {
    if (!forecastTimes || forecastTimes.length === 0) return 0
    return forecastTimes[0]
  }, [forecastTimes])

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

  const hoveredDatum = hoverIndex != null ? data[hoverIndex] : null
  const hoveredDKm = hoveredDatum?.dKm ?? null

  return (
    <div className="mt-4">
      <p className="mb-1 text-xs font-medium text-gray-500">
        Elevation profile (m)
      </p>
      <div className="relative">
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
            <Tooltip content={() => null} />
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
              activeDot={false}
            />
          </AreaChart>
        </ResponsiveContainer>
        {hoveredDatum && (
          <div className="pointer-events-none absolute bottom-2 left-12 rounded bg-white/90 px-2 py-1 text-xs text-gray-700 shadow-sm">
            {hoveredDatum.dKm} km &middot; {hoveredDatum.ele} m
            {hoveredDatum.ts != null && startTs > 0 && (
              <>
                {" "}
                &middot; +{fmtElapsed(hoveredDatum.ts - startTs)} &middot;{" "}
                {fmtClock(hoveredDatum.ts)}
              </>
            )}
          </div>
        )}
      </div>
    </div>
  )
})
