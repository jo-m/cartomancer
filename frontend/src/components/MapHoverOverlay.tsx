import { useHoverStore, useHoverValue } from "../hooks/useHoverSync"
import { fmtElapsed, fmtClock } from "../lib/time"

export interface MapHoverOverlayProps {
  hoverStore: ReturnType<typeof useHoverStore>
  trackPoints: { lat: number; lon: number; ele: number; d: number }[]
  forecastTimes?: number[]
}

/** Overlay on the map showing hover info with forecast timing. */
export default function MapHoverOverlay({
  hoverStore,
  trackPoints,
  forecastTimes,
}: MapHoverOverlayProps) {
  const hoverIndex = useHoverValue(hoverStore)

  if (hoverIndex == null || hoverIndex < 0 || hoverIndex >= trackPoints.length)
    return null

  const p = trackPoints[hoverIndex]
  const dKm = (p.d / 1000).toFixed(1)

  let timeInfo = ""
  if (forecastTimes && forecastTimes.length > hoverIndex) {
    const ts = forecastTimes[hoverIndex]
    const startTs = forecastTimes[0]
    timeInfo = ` · +${fmtElapsed(ts - startTs)} · ${fmtClock(ts)}`
  }

  return (
    <div className="pointer-events-none absolute bottom-2 left-2 rounded bg-panel/90 px-2 py-1 text-xs text-text-secondary shadow-sm">
      {dKm} km &middot; {Math.round(p.ele)} m{timeInfo}
    </div>
  )
}
