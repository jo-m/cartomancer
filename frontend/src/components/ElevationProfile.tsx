import { useEffect, useRef, useCallback, useMemo, memo } from "react"
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
} from "recharts"
import type { HoverStore } from "../hooks/useHoverSync"

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

const Y_AXIS_WIDTH = 44
const CHART_MARGIN = { top: 5, right: 5, bottom: 5, left: 5 }

/** Renders an interactive elevation profile chart using recharts. */
export default memo(function ElevationProfile({
  points,
  hoverStore,
  color,
  forecastTimes,
}: Props) {
  const lineRef = useRef<HTMLDivElement>(null)
  const labelRef = useRef<HTMLDivElement>(null)

  const data: ElevDatum[] = useMemo(
    () =>
      points.map((p, i) => ({
        dKm: Math.round((p.d / 1000) * 100) / 100,
        ele: Math.round(p.ele),
        ts: forecastTimes?.[i] ?? null,
      })),
    [points, forecastTimes]
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

  /** Computes the x pixel position for a given dKm value within the container. */
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

  /** Finds the nearest data index for a mouse x position. */
  const findNearest = useCallback(
    (mouseX: number, containerWidth: number): number | null => {
      if (data.length < 2) return null
      const plotLeft = CHART_MARGIN.left + Y_AXIS_WIDTH
      const plotRight = containerWidth - CHART_MARGIN.right
      if (mouseX < plotLeft || mouseX > plotRight) return null
      const fraction = (mouseX - plotLeft) / (plotRight - plotLeft)
      const minDKm = data[0].dKm
      const maxDKm = data[data.length - 1].dKm
      const dKm = minDKm + fraction * (maxDKm - minDKm)
      let best = 0
      let bestDist = Math.abs(data[0].dKm - dKm)
      for (let i = 1; i < data.length; i++) {
        const dist = Math.abs(data[i].dKm - dKm)
        if (dist < bestDist) {
          bestDist = dist
          best = i
        }
      }
      return best
    },
    [data]
  )

  const updateHoverFromEvent = useCallback(
    (e: React.PointerEvent) => {
      const el = e.currentTarget as HTMLElement
      const rect = el.getBoundingClientRect()
      hoverStore.set(findNearest(e.clientX - rect.left, el.clientWidth))
    },
    [hoverStore, findNearest]
  )

  const handlePointerDown = useCallback(
    (e: React.PointerEvent) => {
      const el = e.currentTarget as HTMLElement
      el.setPointerCapture(e.pointerId)
      const rect = el.getBoundingClientRect()
      const idx = findNearest(e.clientX - rect.left, el.clientWidth)
      // For touch/pen, tapping at the same locked index clears the marker.
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
    [hoverStore, findNearest]
  )

  const handlePointerMove = useCallback(
    (e: React.PointerEvent) => {
      // Mouse: hover-track without buttons. Touch/pen: only while captured (drag).
      if (e.pointerType !== "mouse") {
        const el = e.currentTarget as HTMLElement
        if (!el.hasPointerCapture(e.pointerId)) return
      }
      updateHoverFromEvent(e)
    },
    [updateHoverFromEvent]
  )

  const handlePointerUp = useCallback((e: React.PointerEvent) => {
    const el = e.currentTarget as HTMLElement
    if (el.hasPointerCapture(e.pointerId)) {
      el.releasePointerCapture(e.pointerId)
    }
  }, [])

  const handlePointerLeave = useCallback(
    (e: React.PointerEvent) => {
      // Touch lock persists; mouse leaving clears the marker.
      if (e.pointerType === "mouse") hoverStore.set(null)
    },
    [hoverStore]
  )

  // Imperatively update hover line and label from store changes.
  useEffect(() => {
    return hoverStore.subscribe(() => {
      const idx = hoverStore.get()
      const line = lineRef.current
      const label = labelRef.current
      const container = line?.parentElement
      if (!line || !label || !container) return

      if (idx == null || idx < 0 || idx >= data.length) {
        line.style.display = "none"
        label.style.display = "none"
        return
      }

      const datum = data[idx]
      const x = dKmToX(container.clientWidth, datum.dKm)
      if (x == null) {
        line.style.display = "none"
        label.style.display = "none"
        return
      }

      line.style.display = ""
      line.style.left = `${x}px`
      label.style.display = ""
      label.textContent = `${datum.dKm} km \u00b7 ${datum.ele} m`
    })
  }, [hoverStore, data, dKmToX])

  return (
    <div className="mt-4">
      <p className="mb-1 text-xs font-medium text-text-muted">
        Elevation profile (m)
      </p>
      <div className="relative">
        <ResponsiveContainer width="100%" height={180}>
          <AreaChart data={data} margin={CHART_MARGIN}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
            <XAxis
              dataKey="dKm"
              type="number"
              domain={["dataMin", "dataMax"]}
              tickFormatter={(v: number) => `${v}`}
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
              domain={[minEle, maxEle]}
              tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
              stroke="var(--color-border)"
              width={Y_AXIS_WIDTH}
            />
            <Area
              type="monotone"
              dataKey="ele"
              stroke={color}
              strokeWidth={1.5}
              fill={color}
              fillOpacity={0.1}
              dot={false}
              activeDot={false}
              isAnimationActive={false}
            />
          </AreaChart>
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
          ref={lineRef}
          className="pointer-events-none absolute top-0 bottom-0 w-px bg-text-muted"
          style={{ display: "none" }}
        />
        <div
          ref={labelRef}
          className="pointer-events-none absolute bottom-2 left-12 rounded bg-panel/90 px-2 py-1 text-xs text-text-secondary shadow-sm"
          style={{ display: "none" }}
        />
      </div>
    </div>
  )
})
