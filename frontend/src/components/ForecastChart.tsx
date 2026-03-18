import { useCallback, useMemo } from "react"
import {
  ResponsiveContainer,
  ComposedChart,
  Line,
  Area,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
} from "recharts"
import { useHoverValue, type HoverStore } from "../hooks/useHoverSync"

export interface ForecastPoint {
  index: number
  distanceM: number
  lat: number
  lon: number
  time: string
  temperatureC: number | null
  precipitationRate: number | null
  windSpeedMs: number | null
  windDirectionDeg: number | null
  relativeWindDirectionDeg: number | null
}

interface ChartDatum {
  index: number
  ts: number
  dKm: number
  temperatureC: number | null
  precipitationRate: number | null
  windSpeedMs: number | null
  windDirectionDeg: number | null
  relativeWindDirectionDeg: number | null
  headwindMs: number | null
}

export interface ForecastUnits {
  temperatureC: string
  precipitationRate: string
  windSpeedMs: string
  windDirectionDeg: string
  relativeWindDirectionDeg: string
}

interface Props {
  points: ForecastPoint[]
  units: ForecastUnits
  hoverStore: HoverStore
  attribution?: string
  attributionHref?: string
}

const WIND_DIRS = ["N", "NE", "E", "SE", "S", "SW", "W", "NW"] as const

/** Returns a cardinal direction label for a meteorological wind direction in degrees. */
function windDirLabel(deg: number): string {
  const idx = Math.round(deg / 45) % 8
  return WIND_DIRS[idx]
}

const REL_WIND_LABELS = [
  "Headwind",
  "Head-right",
  "Crosswind R",
  "Tail-right",
  "Tailwind",
  "Tail-left",
  "Crosswind L",
  "Head-left",
] as const

/** Returns a human label for a relative wind direction (0 = headwind, 180 = tailwind). */
function relWindLabel(deg: number): string {
  const idx = Math.round(deg / 45) % 8
  return REL_WIND_LABELS[idx]
}

/** Computes the headwind component (positive = headwind, negative = tailwind). */
function headwindComponent(
  windSpeedMs: number,
  relativeWindDeg: number
): number {
  return windSpeedMs * Math.cos((relativeWindDeg * Math.PI) / 180)
}

/** Renders temperature, precipitation, and wind as vertically stacked recharts. */
export default function ForecastChart({
  points,
  units,
  hoverStore,
  attribution,
  attributionHref,
}: Props) {
  const hoverIndex = useHoverValue(hoverStore)

  const data: ChartDatum[] = useMemo(
    () =>
      points.map((p) => {
        const hw =
          p.windSpeedMs != null && p.relativeWindDirectionDeg != null
            ? Math.round(
                headwindComponent(p.windSpeedMs, p.relativeWindDirectionDeg) *
                  10
              ) / 10
            : null
        return {
          index: p.index,
          ts: new Date(p.time).getTime(),
          dKm: Math.round((p.distanceM / 1000) * 100) / 100,
          temperatureC:
            p.temperatureC != null
              ? Math.round(p.temperatureC * 10) / 10
              : null,
          precipitationRate:
            p.precipitationRate != null
              ? Math.round(p.precipitationRate * 100) / 100
              : null,
          windSpeedMs:
            p.windSpeedMs != null ? Math.round(p.windSpeedMs * 10) / 10 : null,
          windDirectionDeg:
            p.windDirectionDeg != null ? Math.round(p.windDirectionDeg) : null,
          relativeWindDirectionDeg:
            p.relativeWindDirectionDeg != null
              ? Math.round(p.relativeWindDirectionDeg)
              : null,
          headwindMs: hw,
        }
      }),
    [points]
  )

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
    if (temps.length === 0) return [0, 20]
    const lo = Math.min(...temps)
    const hi = Math.max(...temps)
    const mid = (lo + hi) / 2
    // 20-degree range centered on the data midpoint, snapped to 5-degree boundaries.
    const bottom = Math.floor((mid - 10) / 5) * 5
    const top = bottom + 20
    return [bottom, top]
  }, [data])

  const tempTicks = useMemo(() => {
    const ticks: number[] = []
    for (let v = minTemp; v <= maxTemp; v += 5) ticks.push(v)
    return ticks
  }, [minTemp, maxTemp])

  const headwindDomain = useMemo(() => {
    const vals = data
      .map((d) => d.headwindMs)
      .filter((v): v is number => v != null && isFinite(v))
    if (vals.length === 0) return [-5, 5]
    const lo = Math.floor(Math.min(...vals)) - 1
    const hi = Math.ceil(Math.max(...vals)) + 1
    return [Math.min(lo, -1), Math.max(hi, 1)]
  }, [data])

  // Gradient offset for the headwind area chart: fraction from top where y=0 sits.
  const headwindGradientOffset = useMemo(() => {
    const [lo, hi] = headwindDomain
    if (hi <= 0) return 1
    if (lo >= 0) return 0
    return hi / (hi - lo)
  }, [headwindDomain])

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

  const hasRelativeWind = data.some((d) => d.headwindMs != null)

  return (
    <div className="mt-4 space-y-2">
      <div>
        <p className="mb-1 text-xs font-medium text-gray-500">
          Temperature ({units.temperatureC})
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
                ticks={tempTicks}
                allowDataOverflow
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
              {nearestForecast.dKm} km &middot; {nearestForecast.temperatureC}{" "}
              {units.temperatureC}
            </div>
          )}
        </div>
      </div>

      <div>
        <p className="mb-1 text-xs font-medium text-gray-500">
          Precipitation ({units.precipitationRate})
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
                domain={[0, 8]}
                ticks={[0, 2, 4, 6, 8]}
                allowDataOverflow
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
              {nearestForecast.dKm} km &middot;{" "}
              {nearestForecast.precipitationRate} {units.precipitationRate}
            </div>
          )}
        </div>
      </div>

      {hasRelativeWind ? (
        <div>
          <p className="mb-1 text-xs font-medium text-gray-500">
            Head/tailwind ({units.windSpeedMs})
          </p>
          <div className="relative">
            <ResponsiveContainer width="100%" height={140}>
              <ComposedChart
                data={data}
                onMouseMove={handleMouseMove}
                onMouseLeave={handleMouseLeave}
                syncId="forecast"
              >
                <defs>
                  <linearGradient id="headwindGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset={0} stopColor="#ef4444" stopOpacity={0.4} />
                    <stop
                      offset={headwindGradientOffset}
                      stopColor="#ef4444"
                      stopOpacity={0.2}
                    />
                    <stop
                      offset={headwindGradientOffset}
                      stopColor="#10b981"
                      stopOpacity={0.2}
                    />
                    <stop offset={1} stopColor="#10b981" stopOpacity={0.4} />
                  </linearGradient>
                </defs>
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
                  domain={headwindDomain}
                  tick={{ fontSize: 11, fill: "#6b7280" }}
                  stroke="#d1d5db"
                  width={36}
                />
                <Tooltip content={() => null} />
                <ReferenceLine
                  y={0}
                  stroke="#9ca3af"
                  strokeDasharray="4 2"
                  strokeWidth={0.5}
                />
                {referenceDKm != null && (
                  <ReferenceLine
                    x={referenceDKm}
                    stroke="#9ca3af"
                    strokeWidth={1}
                  />
                )}
                <Area
                  type="monotone"
                  dataKey="headwindMs"
                  stroke="none"
                  fill="url(#headwindGrad)"
                  baseValue={0}
                  isAnimationActive={false}
                />
                <Line
                  type="monotone"
                  dataKey="headwindMs"
                  stroke="#6b7280"
                  strokeWidth={1}
                  dot={false}
                  activeDot={false}
                />
              </ComposedChart>
            </ResponsiveContainer>
            {nearestForecast &&
              nearestForecast.headwindMs != null &&
              nearestForecast.relativeWindDirectionDeg != null && (
                <div className="pointer-events-none absolute bottom-2 left-10 rounded bg-white/90 px-2 py-1 text-xs text-gray-700 shadow-sm">
                  {nearestForecast.dKm} km &middot;{" "}
                  {nearestForecast.headwindMs > 0 ? "+" : ""}
                  {nearestForecast.headwindMs} {units.windSpeedMs}{" "}
                  {relWindLabel(nearestForecast.relativeWindDirectionDeg)}
                  {nearestForecast.windSpeedMs != null && (
                    <span className="text-gray-400">
                      {" "}
                      ({nearestForecast.windSpeedMs} {units.windSpeedMs}{" "}
                      {nearestForecast.windDirectionDeg != null &&
                        windDirLabel(nearestForecast.windDirectionDeg)}
                      )
                    </span>
                  )}
                </div>
              )}
          </div>
        </div>
      ) : (
        <div>
          <p className="mb-1 text-xs font-medium text-gray-500">
            Wind ({units.windSpeedMs})
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
                <Line
                  type="monotone"
                  dataKey="windSpeedMs"
                  stroke="#10b981"
                  strokeWidth={1.5}
                  dot={false}
                  activeDot={false}
                />
              </ComposedChart>
            </ResponsiveContainer>
            {nearestForecast &&
              nearestForecast.windSpeedMs != null &&
              nearestForecast.windDirectionDeg != null && (
                <div className="pointer-events-none absolute bottom-2 left-10 rounded bg-white/90 px-2 py-1 text-xs text-gray-700 shadow-sm">
                  {nearestForecast.dKm} km &middot;{" "}
                  {nearestForecast.windSpeedMs} {units.windSpeedMs}{" "}
                  {windDirLabel(nearestForecast.windDirectionDeg)}
                </div>
              )}
          </div>
        </div>
      )}

      {hasRelativeWind && (
        <div className="flex flex-wrap gap-6">
          <WindRose
            points={points}
            config={{
              directionKey: "relativeWindDirectionDeg",
              labels: RELATIVE_ROSE_LABELS,
              sectorColor: relativeSectorColor,
              title: "Relative wind",
            }}
          />
          <WindRose
            points={points}
            config={{
              directionKey: "windDirectionDeg",
              labels: COMPASS_ROSE_LABELS,
              sectorColor: compassSectorColor,
              title: "Compass wind",
            }}
          />
        </div>
      )}

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

const RELATIVE_ROSE_LABELS = [
  "Head",
  "",
  "Right",
  "",
  "Tail",
  "",
  "Left",
  "",
] as const
const COMPASS_ROSE_LABELS = ["N", "", "E", "", "S", "", "W", ""] as const

const ROSE_SECTORS = 8
const ROSE_PAD = 30
const ROSE_MAX_R = 50
const ROSE_SIZE = (ROSE_MAX_R + ROSE_PAD) * 2
const ROSE_CX = ROSE_SIZE / 2
const ROSE_CY = ROSE_SIZE / 2

/** Returns a fill color for a relative wind sector (headwind=red, tailwind=green, cross=gray). */
function relativeSectorColor(i: number): string {
  if (i === 0 || i === 1 || i === 7) return "#ef4444"
  if (i === 3 || i === 4 || i === 5) return "#10b981"
  return "#9ca3af"
}

/** Returns a uniform fill color for compass wind sectors. */
function compassSectorColor(): string {
  return "#6366f1"
}

interface WindRoseConfig {
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
function WindRose({
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
        fill="#6b7280"
      >
        {label}
      </text>
    )
  })

  return (
    <div className="flex items-center gap-3">
      <p className="text-xs font-medium text-gray-500 [writing-mode:vertical-lr] rotate-180">
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
          stroke="#e5e7eb"
          strokeWidth={0.5}
        />
        <circle
          cx={ROSE_CX}
          cy={ROSE_CY}
          r={ROSE_MAX_R / 2}
          fill="none"
          stroke="#e5e7eb"
          strokeWidth={0.5}
        />
        {/* Cross hairs. */}
        <line
          x1={ROSE_CX}
          y1={ROSE_CY - ROSE_MAX_R}
          x2={ROSE_CX}
          y2={ROSE_CY + ROSE_MAX_R}
          stroke="#e5e7eb"
          strokeWidth={0.5}
        />
        <line
          x1={ROSE_CX - ROSE_MAX_R}
          y1={ROSE_CY}
          x2={ROSE_CX + ROSE_MAX_R}
          y2={ROSE_CY}
          stroke="#e5e7eb"
          strokeWidth={0.5}
        />
        {petals}
        {labels}
      </svg>
    </div>
  )
}
