import { useCallback, useEffect, useMemo, useRef } from "react"
import {
  ResponsiveContainer,
  ComposedChart,
  Line,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  ReferenceLine,
} from "recharts"
import type { HoverStore } from "../hooks/useHoverSync"
import { externalUrl } from "../lib/externalUrl"

export interface ForecastPoint {
  distanceM: number
  time: string
  temperatureC: number | null
  precipitationRate: number | null
  windSpeedMs: number | null
  windDirectionDeg: number | null
  relativeWindDirectionDeg: number | null
}

export interface SunEvent {
  type: "dawn" | "sunrise" | "sunset" | "dusk"
  time: string
  distanceM: number
}

interface ChartDatum {
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
  trackDistancesM?: number[]
  attribution?: { text: string; href: string }
  sunEvents?: SunEvent[]
}

const Y_AXIS_WIDTH = 44
const CHART_MARGIN = { top: 5, right: 5, bottom: 5, left: 5 }

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
const SUN_EVENT_COLORS: Record<string, string> = {
  dawn: "#f59e0b",
  sunrise: "#f59e0b",
  sunset: "#8b5cf6",
  dusk: "#8b5cf6",
}

const SUN_EVENT_DASH: Record<string, string> = {
  dawn: "4 3",
  sunrise: "",
  sunset: "",
  dusk: "4 3",
}

/** Renders a sun-up or sun-down icon as a custom ReferenceLine label. */
function SunEventIcon({
  viewBox,
  color,
  rising,
}: {
  viewBox?: { x: number; y: number }
  color: string
  rising: boolean
}) {
  if (!viewBox) return null
  const { x } = viewBox
  const cy = 6
  const r = 5
  const rayLen = 3
  const rays = 8
  return (
    <g>
      {/* Horizon line. */}
      <line
        x1={x - r - rayLen}
        y1={cy + r}
        x2={x + r + rayLen}
        y2={cy + r}
        stroke={color}
        strokeWidth={1}
      />
      {/* Sun disc (half-circle above horizon). */}
      <path
        d={`M${x - r},${cy + r} A${r},${r} 0 1,1 ${x + r},${cy + r}`}
        fill={color}
        fillOpacity={0.3}
        stroke={color}
        strokeWidth={1}
      />
      {/* Rays above horizon. */}
      {Array.from({ length: rays }, (_, i) => {
        const angle = (Math.PI * i) / (rays - 1)
        const x1 = x + (r + 1) * Math.cos(Math.PI - angle)
        const y1 = cy + r - (r + 1) * Math.sin(angle)
        const x2 = x + (r + rayLen) * Math.cos(Math.PI - angle)
        const y2 = cy + r - (r + rayLen) * Math.sin(angle)
        return (
          <line
            key={i}
            x1={x1}
            y1={y1}
            x2={x2}
            y2={y2}
            stroke={color}
            strokeWidth={0.8}
          />
        )
      })}
      {/* Arrow indicating direction. */}
      <polyline
        points={
          rising
            ? `${x - 2.5},${cy - r - rayLen + 2} ${x},${cy - r - rayLen - 1} ${x + 2.5},${cy - r - rayLen + 2}`
            : `${x - 2.5},${cy - r - rayLen - 1} ${x},${cy - r - rayLen + 2} ${x + 2.5},${cy - r - rayLen - 1}`
        }
        fill="none"
        stroke={color}
        strokeWidth={1.2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </g>
  )
}

export default function ForecastChart({
  points,
  units,
  hoverStore,
  trackDistancesM,
  attribution,
  sunEvents,
}: Props) {
  const tempLineRef = useRef<HTMLDivElement>(null)
  const tempLabelRef = useRef<HTMLDivElement>(null)
  const precipLineRef = useRef<HTMLDivElement>(null)
  const precipLabelRef = useRef<HTMLDivElement>(null)
  const windLineRef = useRef<HTMLDivElement>(null)
  const windLabelRef = useRef<HTMLDivElement>(null)

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
          ts: new Date(p.time).getTime(),
          dKm: Math.round((p.distanceM / 1000) * 100) / 100,
          temperatureC:
            p.temperatureC != null
              ? Math.round(p.temperatureC * 10) / 10
              : null,
          precipitationRate:
            p.precipitationRate != null && p.precipitationRate > 0
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

  /** Computes the x pixel position for a given dKm within a container. */
  const dKmToX = useCallback(
    (containerWidth: number, dKm: number): number | null => {
      if (data.length < 2) return null
      const minDKm = data[0].dKm
      const maxDKm = data[data.length - 1].dKm
      const range = maxDKm - minDKm
      if (range <= 0) return null
      const plotLeft = CHART_MARGIN.left + Y_AXIS_WIDTH
      const plotRight = containerWidth - CHART_MARGIN.right
      return plotLeft + ((dKm - minDKm) / range) * (plotRight - plotLeft)
    },
    [data]
  )

  /** Finds the nearest forecast datum for a track point index by distance. */
  const findNearestDatum = useCallback(
    (trackIdx: number): ChartDatum | null => {
      if (data.length === 0) return null
      if (
        !trackDistancesM ||
        trackIdx < 0 ||
        trackIdx >= trackDistancesM.length
      ) {
        return null
      }
      const targetKm = trackDistancesM[trackIdx] / 1000
      let best = 0
      let bestDist = Math.abs(data[0].dKm - targetKm)
      for (let i = 1; i < data.length; i++) {
        const dist = Math.abs(data[i].dKm - targetKm)
        if (dist < bestDist) {
          bestDist = dist
          best = i
        }
      }
      return data[best]
    },
    [data, trackDistancesM]
  )

  /** Finds the nearest track point index for a distance in km. */
  const trackIdxForDKm = useCallback(
    (dKm: number): number | null => {
      if (!trackDistancesM || trackDistancesM.length === 0) return null
      const targetM = dKm * 1000
      let best = 0
      let bestDist = Math.abs(trackDistancesM[0] - targetM)
      for (let i = 1; i < trackDistancesM.length; i++) {
        const dist = Math.abs(trackDistancesM[i] - targetM)
        if (dist < bestDist) {
          bestDist = dist
          best = i
        }
      }
      return best
    },
    [trackDistancesM]
  )

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      const el = e.currentTarget as HTMLElement
      const rect = el.getBoundingClientRect()
      const mouseX = e.clientX - rect.left
      const cw = el.clientWidth
      if (data.length < 2) return
      const plotLeft = CHART_MARGIN.left + Y_AXIS_WIDTH
      const plotRight = cw - CHART_MARGIN.right
      if (mouseX < plotLeft || mouseX > plotRight) {
        hoverStore.set(null)
        return
      }
      const fraction = (mouseX - plotLeft) / (plotRight - plotLeft)
      const minDKm = data[0].dKm
      const maxDKm = data[data.length - 1].dKm
      const dKm = minDKm + fraction * (maxDKm - minDKm)
      const idx = trackIdxForDKm(dKm)
      if (idx == null) return
      hoverStore.set(idx)
    },
    [hoverStore, data, trackIdxForDKm]
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

  const sunEventMarkers = useMemo(
    () =>
      (sunEvents ?? []).map((e) => ({
        dKm: Math.round((e.distanceM / 1000) * 100) / 100,
        color: SUN_EVENT_COLORS[e.type] ?? "var(--color-text-muted)",
        dash: SUN_EVENT_DASH[e.type] ?? "",
        rising: e.type === "dawn" || e.type === "sunrise",
      })),
    [sunEvents]
  )

  const hasRelativeWind = data.some((d) => d.headwindMs != null)

  // Imperatively update all hover lines and labels from store changes.
  useEffect(() => {
    const setLine = (
      ref: React.RefObject<HTMLDivElement | null>,
      x: number | null
    ) => {
      if (!ref.current) return
      if (x != null) {
        ref.current.style.display = ""
        ref.current.style.left = `${x}px`
      } else {
        ref.current.style.display = "none"
      }
    }
    const setLabel = (
      ref: React.RefObject<HTMLDivElement | null>,
      text: string | null
    ) => {
      if (!ref.current) return
      if (text) {
        ref.current.style.display = ""
        ref.current.textContent = text
      } else {
        ref.current.style.display = "none"
      }
    }

    return hoverStore.subscribe(() => {
      const idx = hoverStore.get()
      const nearest = idx != null ? findNearestDatum(idx) : null
      const container = tempLineRef.current?.parentElement
      const cw = container?.clientWidth ?? 0
      const x = nearest ? dKmToX(cw, nearest.dKm) : null

      setLine(tempLineRef, x)
      setLine(precipLineRef, x)
      setLine(windLineRef, x)

      // Temperature label.
      setLabel(
        tempLabelRef,
        nearest?.temperatureC != null
          ? `${nearest.dKm} km \u00b7 ${nearest.temperatureC} ${units.temperatureC}`
          : null
      )

      // Precipitation label.
      setLabel(
        precipLabelRef,
        nearest?.precipitationRate != null
          ? `${nearest.dKm} km \u00b7 ${nearest.precipitationRate} ${units.precipitationRate}`
          : null
      )

      // Wind label.
      if (
        hasRelativeWind &&
        nearest?.headwindMs != null &&
        nearest.relativeWindDirectionDeg != null
      ) {
        const sign = nearest.headwindMs > 0 ? "+" : ""
        let text = `${nearest.dKm} km \u00b7 ${sign}${nearest.headwindMs} ${units.windSpeedMs} ${relWindLabel(nearest.relativeWindDirectionDeg)}`
        if (nearest.windSpeedMs != null && nearest.windDirectionDeg != null) {
          text += ` (${nearest.windSpeedMs} ${units.windSpeedMs} ${windDirLabel(nearest.windDirectionDeg)})`
        }
        setLabel(windLabelRef, text)
      } else if (
        !hasRelativeWind &&
        nearest?.windSpeedMs != null &&
        nearest.windDirectionDeg != null
      ) {
        setLabel(
          windLabelRef,
          `${nearest.dKm} km \u00b7 ${nearest.windSpeedMs} ${units.windSpeedMs} ${windDirLabel(nearest.windDirectionDeg)}`
        )
      } else {
        setLabel(windLabelRef, null)
      }
    })
  }, [hoverStore, data, units, hasRelativeWind, findNearestDatum, dKmToX])

  return (
    <div className="mt-4 space-y-2">
      <div>
        <p className="mb-1 text-xs font-medium text-text-muted">
          Temperature ({units.temperatureC})
        </p>
        <div className="relative">
          <ResponsiveContainer width="100%" height={180}>
            <ComposedChart data={data} margin={CHART_MARGIN}>
              <CartesianGrid
                strokeDasharray="3 3"
                stroke="var(--color-border)"
              />
              <XAxis
                dataKey="dKm"
                type="number"
                domain={["dataMin", "dataMax"]}
                tickFormatter={xTickFormatter}
                tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
                stroke="var(--color-border)"
                label={{
                  value: "km",
                  position: "insideBottomRight",
                  offset: -5,
                  style: { fontSize: 10, fill: "var(--color-text-muted)" },
                }}
              />
              <YAxis
                domain={[minTemp, maxTemp]}
                ticks={tempTicks}
                allowDataOverflow
                tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
                stroke="var(--color-border)"
                width={Y_AXIS_WIDTH}
              />
              <Line
                type="monotone"
                dataKey="temperatureC"
                stroke="#ef4444"
                strokeWidth={1.5}
                dot={false}
                activeDot={false}
                isAnimationActive={false}
              />
              {sunEventMarkers.map((m, i) => (
                <ReferenceLine
                  key={i}
                  x={m.dKm}
                  stroke={m.color}
                  strokeWidth={1}
                  strokeDasharray={m.dash}
                  label={<SunEventIcon color={m.color} rising={m.rising} />}
                />
              ))}
            </ComposedChart>
          </ResponsiveContainer>
          <div
            className="absolute inset-0 cursor-crosshair"
            onMouseMove={handleMouseMove}
            onMouseLeave={handleMouseLeave}
          />
          <div
            ref={tempLineRef}
            className="pointer-events-none absolute top-0 bottom-0 w-px bg-text-muted"
            style={{ display: "none" }}
          />
          <div
            ref={tempLabelRef}
            className="pointer-events-none absolute bottom-2 left-12 rounded bg-panel/90 px-2 py-1 text-xs text-text-secondary shadow-sm"
            style={{ display: "none" }}
          />
        </div>
      </div>

      <div>
        <p className="mb-1 text-xs font-medium text-text-muted">
          Precipitation ({units.precipitationRate})
        </p>
        <div className="relative">
          <ResponsiveContainer width="100%" height={120}>
            <ComposedChart data={data} margin={CHART_MARGIN}>
              <CartesianGrid
                strokeDasharray="3 3"
                stroke="var(--color-border)"
              />
              <XAxis
                dataKey="dKm"
                type="number"
                domain={["dataMin", "dataMax"]}
                tickFormatter={xTickFormatter}
                tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
                stroke="var(--color-border)"
                label={{
                  value: "km",
                  position: "insideBottomRight",
                  offset: -5,
                  style: { fontSize: 10, fill: "var(--color-text-muted)" },
                }}
              />
              <YAxis
                domain={[0, 8]}
                ticks={[0, 2, 4, 6, 8]}
                allowDataOverflow
                tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
                stroke="var(--color-border)"
                width={Y_AXIS_WIDTH}
              />
              <Area
                type="stepAfter"
                dataKey="precipitationRate"
                fill="#3b82f6"
                fillOpacity={0.5}
                stroke="#3b82f6"
                strokeWidth={1}
                dot={false}
                activeDot={false}
                isAnimationActive={false}
                connectNulls={false}
              />
              {sunEventMarkers.map((m, i) => (
                <ReferenceLine
                  key={i}
                  x={m.dKm}
                  stroke={m.color}
                  strokeWidth={1}
                  strokeDasharray={m.dash}
                />
              ))}
            </ComposedChart>
          </ResponsiveContainer>
          <div
            className="absolute inset-0 cursor-crosshair"
            onMouseMove={handleMouseMove}
            onMouseLeave={handleMouseLeave}
          />
          <div
            ref={precipLineRef}
            className="pointer-events-none absolute top-0 bottom-0 w-px bg-text-muted"
            style={{ display: "none" }}
          />
          <div
            ref={precipLabelRef}
            className="pointer-events-none absolute bottom-2 left-12 rounded bg-panel/90 px-2 py-1 text-xs text-text-secondary shadow-sm"
            style={{ display: "none" }}
          />
        </div>
      </div>

      {hasRelativeWind ? (
        <div>
          <p className="mb-1 text-xs font-medium text-text-muted">
            Head/tailwind ({units.windSpeedMs})
          </p>
          <div className="relative">
            <ResponsiveContainer width="100%" height={140}>
              <ComposedChart data={data} margin={CHART_MARGIN}>
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
                <CartesianGrid
                  strokeDasharray="3 3"
                  stroke="var(--color-border)"
                />
                <XAxis
                  dataKey="dKm"
                  type="number"
                  domain={["dataMin", "dataMax"]}
                  tickFormatter={xTickFormatter}
                  tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
                  stroke="var(--color-border)"
                  label={{
                    value: "km",
                    position: "insideBottomRight",
                    offset: -5,
                    style: { fontSize: 10, fill: "var(--color-text-muted)" },
                  }}
                />
                <YAxis
                  domain={headwindDomain}
                  tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
                  stroke="var(--color-border)"
                  width={Y_AXIS_WIDTH}
                />
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
                  stroke="var(--color-text-muted)"
                  strokeWidth={1}
                  dot={false}
                  activeDot={false}
                  isAnimationActive={false}
                />
                {sunEventMarkers.map((m, i) => (
                  <ReferenceLine
                    key={i}
                    x={m.dKm}
                    stroke={m.color}
                    strokeWidth={1}
                    strokeDasharray={m.dash}
                  />
                ))}
              </ComposedChart>
            </ResponsiveContainer>
            <div
              className="absolute inset-0 cursor-crosshair"
              onMouseMove={handleMouseMove}
              onMouseLeave={handleMouseLeave}
            />
            <div
              ref={windLineRef}
              className="pointer-events-none absolute top-0 bottom-0 w-px bg-text-muted"
              style={{ display: "none" }}
            />
            <div
              ref={windLabelRef}
              className="pointer-events-none absolute bottom-2 left-12 rounded bg-panel/90 px-2 py-1 text-xs text-text-secondary shadow-sm"
              style={{ display: "none" }}
            />
          </div>
        </div>
      ) : (
        <div>
          <p className="mb-1 text-xs font-medium text-text-muted">
            Wind ({units.windSpeedMs})
          </p>
          <div className="relative">
            <ResponsiveContainer width="100%" height={120}>
              <ComposedChart data={data} margin={CHART_MARGIN}>
                <CartesianGrid
                  strokeDasharray="3 3"
                  stroke="var(--color-border)"
                />
                <XAxis
                  dataKey="dKm"
                  type="number"
                  domain={["dataMin", "dataMax"]}
                  tickFormatter={xTickFormatter}
                  tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
                  stroke="var(--color-border)"
                  label={{
                    value: "km",
                    position: "insideBottomRight",
                    offset: -5,
                    style: { fontSize: 10, fill: "var(--color-text-muted)" },
                  }}
                />
                <YAxis
                  tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
                  stroke="var(--color-border)"
                  width={Y_AXIS_WIDTH}
                />
                <Line
                  type="monotone"
                  dataKey="windSpeedMs"
                  stroke="#10b981"
                  strokeWidth={1.5}
                  dot={false}
                  activeDot={false}
                  isAnimationActive={false}
                />
                {sunEventMarkers.map((m, i) => (
                  <ReferenceLine
                    key={i}
                    x={m.dKm}
                    stroke={m.color}
                    strokeWidth={1}
                    strokeDasharray={m.dash}
                  />
                ))}
              </ComposedChart>
            </ResponsiveContainer>
            <div
              className="absolute inset-0 cursor-crosshair"
              onMouseMove={handleMouseMove}
              onMouseLeave={handleMouseLeave}
            />
            <div
              ref={windLineRef}
              className="pointer-events-none absolute top-0 bottom-0 w-px bg-text-muted"
              style={{ display: "none" }}
            />
            <div
              ref={windLabelRef}
              className="pointer-events-none absolute bottom-2 left-12 rounded bg-panel/90 px-2 py-1 text-xs text-text-secondary shadow-sm"
              style={{ display: "none" }}
            />
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
        <p className="mt-1 text-right text-[10px] text-text-muted">
          Source:{" "}
          {attribution.href ? (
            <a
              href={externalUrl(attribution.href)}
              target="_blank"
              rel="noopener noreferrer"
              className="underline hover:text-text-secondary transition-colors"
            >
              {attribution.text}
            </a>
          ) : (
            attribution.text
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
