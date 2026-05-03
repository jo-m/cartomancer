import Alert from "./ui/Alert"
import { fmtClock } from "../lib/time"

const START_HOUR_OPTIONS = [1, 2, 5, 10, 20]
const SPEED_OPTIONS = [20, 25, 28, 30]

export interface ForecastControlsProps {
  startHoursOffset: number
  speedKmh: number
  estDurationH: number
  forecastLoading: boolean
  forecastStatus: string | null
  getStartTime: (hoursOffset: number) => Date
  onChangeStart: (h: number) => void
  onChangeSpeed: (s: number) => void
}

/** Controls for adjusting forecast start time and estimated speed. */
export default function ForecastControls({
  startHoursOffset,
  speedKmh,
  estDurationH,
  forecastLoading,
  forecastStatus,
  getStartTime,
  onChangeStart,
  onChangeSpeed,
}: ForecastControlsProps) {
  return (
    <div className="mt-3">
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-1">
          <span className="text-xs text-text-muted">Start in:</span>
          {START_HOUR_OPTIONS.map((h) => (
            <button
              key={h}
              type="button"
              onClick={() => onChangeStart(h)}
              className={`inline-flex min-h-11 cursor-pointer items-center rounded border px-2.5 py-1 text-xs transition-colors ${
                startHoursOffset === h
                  ? "border-border-hover bg-surface font-medium text-text"
                  : "border-border text-text-secondary hover:bg-surface"
              }`}
            >
              +{h}h
            </button>
          ))}
        </div>
        <div className="flex items-center gap-1">
          <span className="text-xs text-text-muted">Est. speed:</span>
          {SPEED_OPTIONS.map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => onChangeSpeed(s)}
              className={`inline-flex min-h-11 cursor-pointer items-center rounded border px-2.5 py-1 text-xs transition-colors ${
                speedKmh === s
                  ? "border-border-hover bg-surface font-medium text-text"
                  : "border-border text-text-secondary hover:bg-surface"
              }`}
            >
              {s}km/h
            </button>
          ))}
        </div>
        {forecastLoading && (
          <span className="text-xs text-text-muted">Loading...</span>
        )}
        {!forecastLoading && estDurationH > 0 && startHoursOffset > 0 && (
          <span className="text-xs text-text-muted">
            Est. {estDurationH.toFixed(1)}h,&ensp;
            {fmtClock(getStartTime(startHoursOffset).getTime())}-
            {fmtClock(getStartTime(startHoursOffset + estDurationH).getTime())}
          </span>
        )}
      </div>
      {forecastStatus === "none" && (
        <Alert variant="warning" className="mt-2 text-xs">
          No weather forecast data available. Time estimates are still shown.
        </Alert>
      )}
      {forecastStatus === "partial" && (
        <Alert variant="warning" className="mt-2 text-xs">
          Weather forecast only partially covers the requested time window.
        </Alert>
      )}
    </div>
  )
}
