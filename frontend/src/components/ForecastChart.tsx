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
import {
  compassSectorColor,
  headwindComponent,
  relativeSectorColor,
  relWindLabel,
  windDirLabel,
} from "../lib/forecast"
import type {
  ForecastPoint,
  ForecastUnits,
  SunEvent,
  SunIntensity as SunIntensityValue,
} from "../types/forecast"
import { findNearestIndex } from "../lib/nearest"
import WindRose from "./WindRose"
import SunIntensity from "./SunIntensity"

interface ChartDatum {
  ts: number
  dKm: number
  temperatureC: number | null
  precipitationRate: number | null
  windSpeedMs: number | null
  windDirectionDeg: number | null
  relativeWindDirectionDeg: number | null
  headwindMs: number | null
  solarRadiationWm2: number | null
}

interface Props {
  points: ForecastPoint[]
  units: ForecastUnits
  hoverStore: HoverStore
  trackDistancesM?: number[]
  attribution?: { text: string; href: string }
  sunEvents?: SunEvent[]
  sunIntensity?: SunIntensityValue | null
}

const Y_AXIS_WIDTH = 44
const CHART_MARGIN = { top: 5, right: 5, bottom: 5, left: 5 }

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

/** Renders temperature, precipitation, and wind as vertically stacked recharts. */
export default function ForecastChart({
  points,
  units,
  hoverStore,
  trackDistancesM,
  attribution,
  sunEvents,
  sunIntensity,
}: Props) {
  const tempLineRef = useRef<HTMLDivElement>(null)
  const tempLabelRef = useRef<HTMLDivElement>(null)
  const precipLineRef = useRef<HTMLDivElement>(null)
  const precipLabelRef = useRef<HTMLDivElement>(null)
  const windLineRef = useRef<HTMLDivElement>(null)
  const windLabelRef = useRef<HTMLDivElement>(null)
  const solarLineRef = useRef<HTMLDivElement>(null)
  const solarLabelRef = useRef<HTMLDivElement>(null)

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
          solarRadiationWm2:
            p.solarRadiationWm2 != null
              ? Math.round(p.solarRadiationWm2)
              : null,
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
      const idx = findNearestIndex(data, targetKm, (d) => d.dKm)
      return idx >= 0 ? data[idx] : null
    },
    [data, trackDistancesM]
  )

  /** Finds the nearest track point index for a distance in km. */
  const trackIdxForDKm = useCallback(
    (dKm: number): number | null => {
      if (!trackDistancesM || trackDistancesM.length === 0) return null
      const targetM = dKm * 1000
      return findNearestIndex(trackDistancesM, targetM, (m) => m)
    },
    [trackDistancesM]
  )

  /** Returns the track index nearest to a pointer x within an overlay element. */
  const idxFromEvent = useCallback(
    (e: React.PointerEvent): number | null => {
      if (data.length < 2) return null
      const el = e.currentTarget as HTMLElement
      const rect = el.getBoundingClientRect()
      const mouseX = e.clientX - rect.left
      const cw = el.clientWidth
      const plotLeft = CHART_MARGIN.left + Y_AXIS_WIDTH
      const plotRight = cw - CHART_MARGIN.right
      if (mouseX < plotLeft || mouseX > plotRight) return null
      const fraction = (mouseX - plotLeft) / (plotRight - plotLeft)
      const minDKm = data[0].dKm
      const maxDKm = data[data.length - 1].dKm
      const dKm = minDKm + fraction * (maxDKm - minDKm)
      return trackIdxForDKm(dKm)
    },
    [data, trackIdxForDKm]
  )

  const handlePointerDown = useCallback(
    (e: React.PointerEvent) => {
      const el = e.currentTarget as HTMLElement
      el.setPointerCapture(e.pointerId)
      const idx = idxFromEvent(e)
      if (
        e.pointerType !== "mouse" &&
        idx != null &&
        hoverStore.get() === idx
      ) {
        hoverStore.set(null)
      } else {
        hoverStore.set(idx)
      }
    },
    [hoverStore, idxFromEvent]
  )

  const handlePointerMove = useCallback(
    (e: React.PointerEvent) => {
      if (e.pointerType !== "mouse") {
        const el = e.currentTarget as HTMLElement
        if (!el.hasPointerCapture(e.pointerId)) return
      }
      const idx = idxFromEvent(e)
      if (idx == null && e.pointerType === "mouse") {
        hoverStore.set(null)
        return
      }
      if (idx != null) hoverStore.set(idx)
    },
    [hoverStore, idxFromEvent]
  )

  const handlePointerUp = useCallback((e: React.PointerEvent) => {
    const el = e.currentTarget as HTMLElement
    if (el.hasPointerCapture(e.pointerId)) {
      el.releasePointerCapture(e.pointerId)
    }
  }, [])

  const handlePointerLeave = useCallback(
    (e: React.PointerEvent) => {
      if (e.pointerType === "mouse") hoverStore.set(null)
    },
    [hoverStore]
  )

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

  const hasSolar = data.some((d) => d.solarRadiationWm2 != null)

  const solarMax = useMemo(() => {
    const vals = data
      .map((d) => d.solarRadiationWm2)
      .filter((v): v is number => v != null && isFinite(v))
    if (vals.length === 0) return 1000
    const hi = Math.max(...vals)
    // Round up to next 200 W/m² boundary, with a 1000 W/m² floor so the
    // y-axis stays comparable across rides.
    return Math.max(1000, Math.ceil(hi / 200) * 200)
  }, [data])

  const solarTicks = useMemo(() => {
    const ticks: number[] = []
    for (let v = 0; v <= solarMax; v += 200) ticks.push(v)
    return ticks
  }, [solarMax])

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
      setLine(solarLineRef, x)

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

      // Solar radiation label.
      setLabel(
        solarLabelRef,
        nearest?.solarRadiationWm2 != null
          ? `${nearest.dKm} km · ${nearest.solarRadiationWm2} ${units.solarRadiationWm2}`
          : null
      )
    })
  }, [hoverStore, data, units, hasRelativeWind, findNearestDatum, dKmToX])

  return (
    <div className="mt-4 space-y-2">
      {hasSolar && (
        <div>
          <p className="mb-1 text-xs font-medium text-text-muted">
            Solar radiation ({units.solarRadiationWm2})
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
                  domain={[0, solarMax]}
                  ticks={solarTicks}
                  allowDataOverflow
                  tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
                  stroke="var(--color-border)"
                  width={Y_AXIS_WIDTH}
                />
                <Area
                  type="monotone"
                  dataKey="solarRadiationWm2"
                  fill="#f59e0b"
                  fillOpacity={0.4}
                  stroke="#f59e0b"
                  strokeWidth={1.5}
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
              className="absolute inset-0 cursor-crosshair touch-pan-y"
              onPointerDown={handlePointerDown}
              onPointerMove={handlePointerMove}
              onPointerUp={handlePointerUp}
              onPointerCancel={handlePointerUp}
              onPointerLeave={handlePointerLeave}
            />
            <div
              ref={solarLineRef}
              className="pointer-events-none absolute top-0 bottom-0 w-px bg-text-muted"
              style={{ display: "none" }}
            />
            <div
              ref={solarLabelRef}
              className="pointer-events-none absolute bottom-2 left-12 rounded bg-panel/90 px-2 py-1 text-xs text-text-secondary shadow-sm"
              style={{ display: "none" }}
            />
          </div>
        </div>
      )}

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
            className="absolute inset-0 cursor-crosshair touch-pan-y"
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerUp}
            onPointerCancel={handlePointerUp}
            onPointerLeave={handlePointerLeave}
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
            className="absolute inset-0 cursor-crosshair touch-pan-y"
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerUp}
            onPointerCancel={handlePointerUp}
            onPointerLeave={handlePointerLeave}
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
              className="absolute inset-0 cursor-crosshair touch-pan-y"
              onPointerDown={handlePointerDown}
              onPointerMove={handlePointerMove}
              onPointerUp={handlePointerUp}
              onPointerCancel={handlePointerUp}
              onPointerLeave={handlePointerLeave}
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
              className="absolute inset-0 cursor-crosshair touch-pan-y"
              onPointerDown={handlePointerDown}
              onPointerMove={handlePointerMove}
              onPointerUp={handlePointerUp}
              onPointerCancel={handlePointerUp}
              onPointerLeave={handlePointerLeave}
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

      {(hasRelativeWind || sunIntensity != null) && (
        <div className="flex flex-wrap gap-6">
          {hasRelativeWind && (
            <>
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
            </>
          )}
          {sunIntensity != null && (
            <SunIntensity
              value={sunIntensity.index}
              doseJm2={sunIntensity.doseJm2}
            />
          )}
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
