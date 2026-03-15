import { useCallback, useMemo } from "react"
import {
  ResponsiveContainer,
  ComposedChart,
  Line,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
} from "recharts"
import { useHoverValue, type HoverStore } from "../hooks/useHoverSync"
import { fmtElapsed, fmtClock } from "../lib/time"

export interface ForecastPoint {
  index: number
  distanceM: number
  lat: number
  lon: number
  time: string
  temperatureC: number | null
  precipitationRate: number | null
}

interface ChartDatum {
  index: number
  ts: number
  dKm: number
  temperatureC: number | null
  precipitationRate: number | null
}

interface Props {
  points: ForecastPoint[]
  hoverStore: HoverStore
  attribution?: string
  attributionHref?: string
}

/** Renders temperature and precipitation as two vertically stacked recharts. */
export default function ForecastChart({ points, hoverStore, attribution, attributionHref }: Props) {
  const hoverIndex = useHoverValue(hoverStore)

  const data: ChartDatum[] = useMemo(
    () =>
      points.map((p) => ({
        index: p.index,
        ts: new Date(p.time).getTime(),
        dKm: Math.round((p.distanceM / 1000) * 100) / 100,
        temperatureC:
          p.temperatureC != null ? Math.round(p.temperatureC * 10) / 10 : null,
        precipitationRate:
          p.precipitationRate != null
            ? Math.round(p.precipitationRate * 100) / 100
            : null,
      })),
    [points]
  )

  const startTs = data.length > 0 ? data[0].ts : 0

  const handleMouseMove = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (state: any) => {
      if (state?.activeTooltipIndex != null) {
        const idx = state.activeTooltipIndex as number
        if (data[idx]) {
          hoverStore.set(data[idx].index)
        }
      }
    },
    [hoverStore, data]
  )

  const handleMouseLeave = useCallback(() => {
    hoverStore.set(null)
  }, [hoverStore])

  const [minTemp, maxTemp] = useMemo(() => {
    const temps = data
      .map((d) => d.temperatureC)
      .filter((v): v is number => v != null && isFinite(v))
    const lo = temps.length ? Math.floor(Math.min(...temps)) - 1 : 0
    const hi = temps.length ? Math.ceil(Math.max(...temps)) + 1 : 20
    return [lo, hi]
  }, [data])

  const xTickFormatter = (v: number) => `${v}`

  const nearestForecast = useMemo(() => {
    if (hoverIndex == null || data.length === 0) return null
    let best = 0
    let bestDist = Math.abs(data[0].index - hoverIndex)
    for (let i = 1; i < data.length; i++) {
      const dist = Math.abs(data[i].index - hoverIndex)
      if (dist < bestDist) {
        bestDist = dist
        best = i
      }
    }
    return data[best]
  }, [hoverIndex, data])

  const referenceDKm = nearestForecast?.dKm ?? null

  return (
    <div className="mt-4 space-y-2">
      <div>
        <p className="mb-1 text-xs font-medium text-gray-500">
          Temperature (C)
        </p>
        <div className="relative">
          <ResponsiveContainer width="100%" height={180}>
            <ComposedChart
              data={data}
              onMouseMove={handleMouseMove}
              onMouseLeave={handleMouseLeave}
              syncId="forecast"
            >
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
              <XAxis
                dataKey="dKm"
                type="number"
                domain={["dataMin", "dataMax"]}
                tickFormatter={xTickFormatter}
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
                domain={[minTemp, maxTemp]}
                tick={{ fontSize: 11, fill: "#6b7280" }}
                stroke="#d1d5db"
                width={36}
              />
              <Tooltip content={() => null} />
              {minTemp < 0 && maxTemp > 0 && (
                <ReferenceLine
                  y={0}
                  stroke="#9ca3af"
                  strokeDasharray="4 2"
                  strokeWidth={0.5}
                />
              )}
              {referenceDKm != null && (
                <ReferenceLine
                  x={referenceDKm}
                  stroke="#9ca3af"
                  strokeWidth={1}
                />
              )}
              <Line
                type="monotone"
                dataKey="temperatureC"
                stroke="#ef4444"
                strokeWidth={1.5}
                dot={false}
                activeDot={false}
              />
            </ComposedChart>
          </ResponsiveContainer>
          {nearestForecast && nearestForecast.temperatureC != null && (
            <div className="pointer-events-none absolute bottom-2 left-10 rounded bg-white/90 px-2 py-1 text-xs text-gray-700 shadow-sm">
              {nearestForecast.dKm} km &middot; +
              {fmtElapsed(nearestForecast.ts - startTs)} &middot;{" "}
              {fmtClock(nearestForecast.ts)} &middot;{" "}
              {nearestForecast.temperatureC} C
            </div>
          )}
        </div>
      </div>

      <div>
        <p className="mb-1 text-xs font-medium text-gray-500">
          Precipitation rate
        </p>
        <div className="relative">
          <ResponsiveContainer width="100%" height={120}>
            <ComposedChart
              data={data}
              onMouseMove={handleMouseMove}
              onMouseLeave={handleMouseLeave}
              syncId="forecast"
            >
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
              <XAxis
                dataKey="dKm"
                type="number"
                domain={["dataMin", "dataMax"]}
                tickFormatter={xTickFormatter}
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
                tick={{ fontSize: 11, fill: "#6b7280" }}
                stroke="#d1d5db"
                width={36}
              />
              <Tooltip content={() => null} />
              {referenceDKm != null && (
                <ReferenceLine
                  x={referenceDKm}
                  stroke="#9ca3af"
                  strokeWidth={1}
                />
              )}
              <Bar
                dataKey="precipitationRate"
                fill="#3b82f6"
                opacity={0.7}
                maxBarSize={8}
              />
            </ComposedChart>
          </ResponsiveContainer>
          {nearestForecast && nearestForecast.precipitationRate != null && (
            <div className="pointer-events-none absolute bottom-2 left-10 rounded bg-white/90 px-2 py-1 text-xs text-gray-700 shadow-sm">
              {nearestForecast.dKm} km &middot; Precip:{" "}
              {nearestForecast.precipitationRate}
            </div>
          )}
        </div>
      </div>
      {attribution && (
        <p className="mt-1 text-right text-[10px] text-gray-400">
          Source:{" "}
          {attributionHref ? (
            <a
              href={attributionHref}
              target="_blank"
              rel="noopener noreferrer"
              className="underline hover:text-gray-600"
            >
              {attribution}
            </a>
          ) : (
            attribution
          )}
        </p>
      )}
    </div>
  )
}
