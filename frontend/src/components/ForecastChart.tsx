import { useState, useCallback, useMemo } from "react"
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
import { fetchClient } from "../api/client"
import { useHoverValue, type HoverStore } from "../hooks/useHoverSync"

interface ForecastPoint {
  index: number
  distanceM: number
  lat: number
  lon: number
  time: string
  temperatureC: number
  precipitationRate: number
}

interface ChartDatum {
  index: number
  ts: number
  dKm: number
  temperatureC: number
  precipitationRate: number
}

interface Props {
  trackUuid: string
  totalDistanceM: number
  onError: (msg: string) => void
  hoverStore: HoverStore
}

/** Formats a duration in milliseconds as "Xh YYmin". */
function fmtElapsed(ms: number): string {
  const totalMin = Math.round(ms / 60000)
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  if (h === 0) return `${m}min`
  return `${h}h ${m.toString().padStart(2, "0")}min`
}

/** Formats a timestamp as HH:MM in 24-hour format. */
function fmtTime(ts: number): string {
  return new Date(ts).toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}

/** Renders the forecast temperature and precipitation chart. */
export default function ForecastChart({
  trackUuid,
  totalDistanceM,
  onError,
  hoverStore,
}: Props) {
  const [points, setPoints] = useState<ForecastPoint[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [startTime, setStartTime] = useState(() => {
    const d = new Date()
    d.setMinutes(0, 0, 0)
    d.setHours(d.getHours() + 1)
    return d.toISOString().slice(0, 16)
  })
  const [speedKmh, setSpeedKmh] = useState("25")

  /** Sets startTime to now + the given number of hours, rounded to the hour. */
  function setStartInHours(h: number) {
    const d = new Date()
    d.setMinutes(0, 0, 0)
    d.setHours(d.getHours() + h)
    setStartTime(d.toISOString().slice(0, 16))
  }

  async function handleGenerate() {
    const speed = parseFloat(speedKmh)
    if (isNaN(speed) || speed <= 0) {
      onError("Speed must be a positive number")
      return
    }
    const isoTime = new Date(startTime).toISOString()
    setLoading(true)
    try {
      const { data, error } = await fetchClient.POST(
        "/tracks/{uuid}/forecast",
        {
          params: {
            path: { uuid: trackUuid },
            query: { startTime: isoTime, speedKmh: speed },
          },
        }
      )
      if (error) {
        throw new Error((error as { msg?: string }).msg ?? "Forecast failed")
      }
      if (!data?.points?.length) {
        onError("No forecast data returned")
        return
      }
      setPoints(data.points as ForecastPoint[])
    } catch (err) {
      onError((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const speed = parseFloat(speedKmh)
  const estDurationH =
    !isNaN(speed) && speed > 0 ? totalDistanceM / 1000 / speed : 0

  return (
    <div className="mt-6">
      <h2 className="text-xs font-medium uppercase tracking-wide text-gray-500">
        Weather forecast
      </h2>

      <div className="mt-3 flex flex-wrap items-end gap-3">
        <div>
          <label className="block text-xs font-medium text-gray-700">
            Start time
          </label>
          <div className="mt-1 flex items-center gap-1.5">
            <input
              type="datetime-local"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              className="rounded border border-gray-200 px-3 py-1.5 text-sm text-gray-900 focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
            {[1, 2, 5, 12].map((h) => (
              <button
                key={h}
                type="button"
                onClick={() => setStartInHours(h)}
                className="rounded border border-gray-200 px-1.5 py-1.5 text-xs text-gray-600 hover:bg-gray-100"
              >
                +{h}h
              </button>
            ))}
          </div>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-700">
            Avg speed (km/h)
          </label>
          <div className="mt-1 flex items-center gap-1.5">
            <input
              type="number"
              min="1"
              max="200"
              step="0.5"
              value={speedKmh}
              onChange={(e) => setSpeedKmh(e.target.value)}
              className="w-24 rounded border border-gray-200 px-3 py-1.5 text-sm text-gray-900 focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
            {[20, 25, 30].map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => setSpeedKmh(String(s))}
                className="rounded border border-gray-200 px-1.5 py-1.5 text-xs text-gray-600 hover:bg-gray-100"
              >
                {s}
              </button>
            ))}
          </div>
        </div>
        <button
          onClick={handleGenerate}
          disabled={loading}
          className="rounded border border-gray-300 bg-white px-4 py-1.5 text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {loading ? "Loading..." : "Generate forecast"}
        </button>
        {estDurationH > 0 && (
          <span className="text-xs text-gray-500">
            Est. {estDurationH.toFixed(1)} h
          </span>
        )}
      </div>

      {points && <Charts points={points} hoverStore={hoverStore} />}
    </div>
  )
}

/** Renders temperature and precipitation as two vertically stacked recharts. */
function Charts({
  points,
  hoverStore,
}: {
  points: ForecastPoint[]
  hoverStore: HoverStore
}) {
  const hoverIndex = useHoverValue(hoverStore)
  const [activeIndex, setActiveIndex] = useState<number | null>(null)

  const data: ChartDatum[] = useMemo(
    () =>
      points.map((p) => ({
        index: p.index,
        ts: new Date(p.time).getTime(),
        dKm: Math.round((p.distanceM / 1000) * 100) / 100,
        temperatureC: Math.round(p.temperatureC * 10) / 10,
        precipitationRate: Math.round(p.precipitationRate * 100) / 100,
      })),
    [points]
  )

  const startTs = data.length > 0 ? data[0].ts : 0

  const handleMouseMove = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (state: any) => {
      if (state?.activeTooltipIndex != null) {
        const idx = state.activeTooltipIndex as number
        setActiveIndex(idx)
        if (data[idx]) {
          hoverStore.set(data[idx].index)
        }
      }
    },
    [hoverStore, data]
  )

  const handleMouseLeave = useCallback(() => {
    setActiveIndex(null)
    hoverStore.set(null)
  }, [hoverStore])

  const [minTemp, maxTemp] = useMemo(() => {
    const temps = data.map((d) => d.temperatureC).filter(isFinite)
    const lo = temps.length ? Math.floor(Math.min(...temps)) - 1 : 0
    const hi = temps.length ? Math.ceil(Math.max(...temps)) + 1 : 20
    return [lo, hi]
  }, [data])

  const xTickFormatter = (v: number) => `${v}`

  // Build the tooltip content for a given data index.
  const renderTooltipContent = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ({ active, payload }: any) => {
      if (!active || !payload?.length) return null
      const d = payload[0].payload as ChartDatum
      const elapsed = fmtElapsed(d.ts - startTs)
      const clock = fmtTime(d.ts)

      // Gather values from all payload entries.
      const lines: [string, string][] = []
      for (const entry of payload) {
        if (entry.dataKey === "temperatureC") {
          lines.push(["Temperature", `${entry.value} C`])
        } else if (entry.dataKey === "precipitationRate") {
          lines.push(["Precipitation", `${entry.value}`])
        }
      }

      return (
        <div
          className="rounded border border-gray-200 bg-white px-3 py-2 text-xs shadow-sm"
          style={{ fontSize: 12 }}
        >
          <p className="mb-1 font-medium text-gray-700">
            {d.dKm} km &middot; +{elapsed} &middot; {clock}
          </p>
          {lines.map(([label, val]) => (
            <p key={label} className="text-gray-600">
              {label}: {val}
            </p>
          ))}
        </div>
      )
    },
    [startTs]
  )

  const internalDKm = activeIndex != null ? data[activeIndex]?.dKm : null

  let externalDKm: number | null = null
  if (hoverIndex != null && activeIndex == null && data[hoverIndex]) {
    externalDKm = data[hoverIndex].dKm
  }

  const referenceDKm = internalDKm ?? externalDKm

  return (
    <div className="mt-4 space-y-2">
      <div>
        <p className="mb-1 text-xs font-medium text-gray-500">
          Temperature (C)
        </p>
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
            <Tooltip content={renderTooltipContent} />
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
              activeDot={{ r: 3, fill: "#ef4444" }}
            />
          </ComposedChart>
        </ResponsiveContainer>
      </div>

      <div>
        <p className="mb-1 text-xs font-medium text-gray-500">
          Precipitation rate
        </p>
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
            <Tooltip content={renderTooltipContent} />
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
      </div>
    </div>
  )
}
