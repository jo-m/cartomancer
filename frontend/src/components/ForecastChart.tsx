import { useState } from "react"
import { fetchClient } from "../api/client"

interface ForecastPoint {
  distanceM: number
  lat: number
  lon: number
  time: string
  temperatureC: number
  precipitationRate: number
}

interface Props {
  trackUuid: string
  totalDistanceM: number
  onError: (msg: string) => void
}

const WIDTH = 600
const HEIGHT = 200
const PAD_LEFT = 50
const PAD_RIGHT = 20
const PAD_TOP = 20
const PAD_BOTTOM = 40
const INNER_W = WIDTH - PAD_LEFT - PAD_RIGHT
const INNER_H = HEIGHT - PAD_TOP - PAD_BOTTOM

/** Formats a Date as HH:MM in 24-hour format. */
function fmtTime(d: Date): string {
  return d.toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}

/** Builds an SVG polyline points string from data. */
function polyline(
  points: ForecastPoint[],
  minT: number,
  maxT: number,
  getValue: (p: ForecastPoint) => number,
  minV: number,
  maxV: number
): string {
  const rangeV = maxV - minV || 1
  const rangeT = maxT - minT || 1
  return points
    .map((p) => {
      const t = new Date(p.time).getTime()
      const x = PAD_LEFT + ((t - minT) / rangeT) * INNER_W
      const y = PAD_TOP + (1 - (getValue(p) - minV) / rangeV) * INNER_H
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(" ")
}

/** Renders the forecast temperature and precipitation chart as an inline SVG. */
export default function ForecastChart({
  trackUuid,
  totalDistanceM,
  onError,
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

  // Estimate duration for display.
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
          {loading ? "Loading…" : "Generate forecast"}
        </button>
        {estDurationH > 0 && (
          <span className="text-xs text-gray-500">
            Est. {estDurationH.toFixed(1)} h
          </span>
        )}
      </div>

      {points && <Chart points={points} />}
    </div>
  )
}

/** Renders temperature and precipitation as two stacked SVG charts. */
function Chart({ points }: { points: ForecastPoint[] }) {
  const times = points.map((p) => new Date(p.time).getTime())
  const minT = Math.min(...times)
  const maxT = Math.max(...times)

  const temps = points.map((p) => p.temperatureC).filter((v) => isFinite(v))
  const minTemp = temps.length ? Math.floor(Math.min(...temps)) - 1 : 0
  const maxTemp = temps.length ? Math.ceil(Math.max(...temps)) + 1 : 20

  const precips = points.map((p) => p.precipitationRate)
  const maxPrecip = Math.max(0.1, ...precips)

  // Generate ~5 time ticks.
  const timeRange = maxT - minT
  const timeTicks: number[] = []
  if (timeRange > 0) {
    const step = timeRange / 5
    for (let i = 0; i <= 5; i++) {
      timeTicks.push(minT + i * step)
    }
  }

  // Generate ~4 temp ticks.
  const tempRange = maxTemp - minTemp
  const tempStep = Math.ceil(tempRange / 4) || 1
  const tempTicks: number[] = []
  for (let v = minTemp; v <= maxTemp; v += tempStep) {
    tempTicks.push(v)
  }

  const tempLine = polyline(
    points,
    minT,
    maxT,
    (p) => p.temperatureC,
    minTemp,
    maxTemp
  )

  return (
    <div className="mt-4 space-y-4">
      {/* Temperature chart */}
      <div>
        <p className="mb-1 text-xs font-medium text-gray-500">
          Temperature (C)
        </p>
        <svg
          viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
          className="w-full rounded border border-gray-200 bg-white"
        >
          {/* Grid lines */}
          {tempTicks.map((v) => {
            const y =
              PAD_TOP + (1 - (v - minTemp) / (maxTemp - minTemp)) * INNER_H
            return (
              <g key={v}>
                <line
                  x1={PAD_LEFT}
                  x2={WIDTH - PAD_RIGHT}
                  y1={y}
                  y2={y}
                  stroke="#e5e7eb"
                  strokeWidth="0.5"
                />
                <text
                  x={PAD_LEFT - 6}
                  y={y + 3}
                  textAnchor="end"
                  fontSize="9"
                  fill="#6b7280"
                >
                  {v}
                </text>
              </g>
            )
          })}

          {/* Time axis */}
          {timeTicks.map((t) => {
            const x = PAD_LEFT + ((t - minT) / (maxT - minT || 1)) * INNER_W
            return (
              <text
                key={t}
                x={x}
                y={HEIGHT - 8}
                textAnchor="middle"
                fontSize="9"
                fill="#6b7280"
              >
                {fmtTime(new Date(t))}
              </text>
            )
          })}

          {/* Zero line */}
          {minTemp < 0 && maxTemp > 0 && (
            <line
              x1={PAD_LEFT}
              x2={WIDTH - PAD_RIGHT}
              y1={PAD_TOP + (1 - (0 - minTemp) / (maxTemp - minTemp)) * INNER_H}
              y2={PAD_TOP + (1 - (0 - minTemp) / (maxTemp - minTemp)) * INNER_H}
              stroke="#9ca3af"
              strokeWidth="0.5"
              strokeDasharray="4 2"
            />
          )}

          <polyline
            points={tempLine}
            fill="none"
            stroke="#ef4444"
            strokeWidth="1.5"
            strokeLinejoin="round"
            strokeLinecap="round"
          />
        </svg>
      </div>

      {/* Precipitation chart */}
      <div>
        <p className="mb-1 text-xs font-medium text-gray-500">
          Precipitation rate
        </p>
        <svg
          viewBox={`0 0 ${WIDTH} ${HEIGHT / 2 + PAD_BOTTOM}`}
          className="w-full rounded border border-gray-200 bg-white"
        >
          {/* Time axis */}
          {timeTicks.map((t) => {
            const x = PAD_LEFT + ((t - minT) / (maxT - minT || 1)) * INNER_W
            return (
              <text
                key={t}
                x={x}
                y={HEIGHT / 2 + PAD_BOTTOM - 8}
                textAnchor="middle"
                fontSize="9"
                fill="#6b7280"
              >
                {fmtTime(new Date(t))}
              </text>
            )
          })}

          {/* Bars */}
          {points.map((p, i) => {
            const t = new Date(p.time).getTime()
            const x = PAD_LEFT + ((t - minT) / (maxT - minT || 1)) * INNER_W
            const barH =
              (p.precipitationRate / maxPrecip) * (HEIGHT / 2 - PAD_TOP)
            const barW = Math.max(1, INNER_W / points.length - 0.5)
            return (
              <rect
                key={i}
                x={x - barW / 2}
                y={HEIGHT / 2 - barH}
                width={barW}
                height={barH}
                fill="#3b82f6"
                opacity="0.7"
              />
            )
          })}
        </svg>
      </div>
    </div>
  )
}
