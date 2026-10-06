import { useCallback } from "react"
import Alert from "./ui/Alert"
import { useNowMs } from "../hooks/useNowMs"
import {
  dayOffsetOf,
  earliestForecastStart,
  latestForecastStart,
  fmtClock,
} from "../lib/time"

const HOUR_MS = 60 * 60 * 1000

/** Slider granularity, matching the 15-minute snapping of start times. */
const STEP_MS = 15 * 60 * 1000

const QUICK_START_OPTIONS = [
  { label: "Now", hours: 0 },
  { label: "+1h", hours: 1 },
  { label: "+2h", hours: 2 },
]

const SPEED_OPTIONS = [20, 25, 28, 30]

export interface ForecastControlsProps {
  startTime: Date
  speedKmh: number
  estDurationH: number
  forecastLoading: boolean
  forecastStatus: string | null
  onChangeStart: (t: Date) => void
  onChangeSpeed: (s: number) => void
}

/** Controls for adjusting the forecast start time and estimated speed. */
export default function ForecastControls({
  startTime,
  speedKmh,
  estDurationH,
  forecastLoading,
  forecastStatus,
  onChangeStart,
  onChangeSpeed,
}: ForecastControlsProps) {
  const nowMs = useNowMs()
  const startMs = startTime.getTime()
  const dayLabel = dayOffsetOf(startMs, nowMs) >= 1 ? "Tomorrow" : "Today"
  const timeLabel = `${dayLabel} ${fmtClock(startMs)}`

  // Event handlers read the clock at press time; render uses the subscribed
  // nowMs instead.
  const readNow = useCallback(() => Date.now(), [])

  // The slider spans the usable forecast range: from the current quarter-hour
  // to the end of the weather data's horizon (~30 h ahead, which reaches into
  // tomorrow).
  const earliestMs = earliestForecastStart(nowMs)
  const latestMs = latestForecastStart(nowMs)

  function pickQuickStart(hours: number) {
    onChangeStart(new Date(readNow() + hours * HOUR_MS))
  }

  const endMs = startMs + estDurationH * HOUR_MS

  return (
    <div className="mt-3">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <div className="flex items-center gap-1">
          <span className="text-xs text-text-muted">Start:</span>
          {QUICK_START_OPTIONS.map((opt) => {
            // Highlight the preset matching the current start time.
            const active =
              Math.abs(startMs - (nowMs + opt.hours * HOUR_MS)) < STEP_MS
            return (
              <button
                key={opt.hours}
                type="button"
                onClick={() => pickQuickStart(opt.hours)}
                className={`inline-flex min-h-11 cursor-pointer items-center rounded border px-2.5 py-1 text-xs transition-colors ${
                  active
                    ? "border-border-hover bg-surface font-medium text-text"
                    : "border-border text-text-secondary hover:bg-surface"
                }`}
              >
                {opt.label}
              </button>
            )
          })}
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
        {!forecastLoading && estDurationH > 0 && (
          <span className="text-xs text-text-muted">
            Est. {estDurationH.toFixed(1)}h,&ensp;{fmtClock(startMs)}-
            {fmtClock(endMs)}
          </span>
        )}
      </div>
      <div className="mt-1 flex items-center gap-3">
        <input
          type="range"
          min={earliestMs}
          max={latestMs}
          step={STEP_MS}
          value={Math.min(Math.max(startMs, earliestMs), latestMs)}
          onChange={(e) => onChangeStart(new Date(Number(e.target.value)))}
          aria-label="Start time"
          aria-valuetext={timeLabel}
          className="h-11 flex-1 cursor-pointer accent-slider-fill"
        />
        <span className="w-26 shrink-0 text-right text-xs font-medium text-text tabular-nums">
          {timeLabel}
        </span>
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
