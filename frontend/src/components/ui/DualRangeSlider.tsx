import { useCallback, useRef } from "react"

const THUMB_SIZE_PX = 24
const HALF_THUMB_PX = THUMB_SIZE_PX / 2

export interface DualRangeSliderProps {
  absoluteMin: number
  absoluteMax: number
  valueMin: number
  valueMax: number
  step: number
  formatValue: (v: number) => string
  onChange: (min: number, max: number) => void
  labelMin?: string
  labelMax?: string
}

/** A dual-thumb range slider for filtering numeric ranges. */
export default function DualRangeSlider({
  absoluteMin,
  absoluteMax,
  valueMin,
  valueMax,
  step,
  formatValue,
  onChange,
  labelMin = "Range minimum",
  labelMax = "Range maximum",
}: DualRangeSliderProps) {
  const outerRef = useRef<HTMLDivElement>(null)
  const activeThumb = useRef<"min" | "max" | null>(null)

  const range = absoluteMax - absoluteMin || 1

  function valueFromClientX(clientX: number): number {
    if (!outerRef.current) return absoluteMin
    const { left, width } = outerRef.current.getBoundingClientRect()
    const p = Math.max(
      0,
      Math.min(1, (clientX - left - HALF_THUMB_PX) / (width - THUMB_SIZE_PX))
    )
    return Math.round((absoluteMin + p * range) / step) * step
  }

  function pickThumb(v: number): "min" | "max" {
    const dMin = Math.abs(v - valueMin)
    const dMax = Math.abs(v - valueMax)
    if (dMin !== dMax) return dMin < dMax ? "min" : "max"
    return v >= (absoluteMin + absoluteMax) / 2 ? "min" : "max"
  }

  function applyValue(v: number) {
    if (activeThumb.current === "min") onChange(Math.min(v, valueMax), valueMax)
    else if (activeThumb.current === "max")
      onChange(valueMin, Math.max(v, valueMin))
  }

  function onPointerDown(e: React.PointerEvent<HTMLDivElement>) {
    e.currentTarget.setPointerCapture(e.pointerId)
    const v = valueFromClientX(e.clientX)
    activeThumb.current = pickThumb(v)
    applyValue(v)
  }

  function onPointerMove(e: React.PointerEvent<HTMLDivElement>) {
    if (!e.currentTarget.hasPointerCapture(e.pointerId)) return
    applyValue(valueFromClientX(e.clientX))
  }

  function onPointerUp() {
    activeThumb.current = null
  }

  const handleKeyDown = useCallback(
    (thumb: "min" | "max", e: React.KeyboardEvent) => {
      let delta = 0
      if (e.key === "ArrowRight" || e.key === "ArrowUp") delta = step
      else if (e.key === "ArrowLeft" || e.key === "ArrowDown") delta = -step
      else if (e.key === "Home") delta = -Infinity
      else if (e.key === "End") delta = Infinity
      else return

      e.preventDefault()
      if (thumb === "min") {
        let next =
          delta === -Infinity
            ? absoluteMin
            : delta === Infinity
              ? valueMax
              : valueMin + delta
        next = Math.max(absoluteMin, Math.min(next, valueMax))
        onChange(next, valueMax)
      } else {
        let next =
          delta === -Infinity
            ? valueMin
            : delta === Infinity
              ? absoluteMax
              : valueMax + delta
        next = Math.max(valueMin, Math.min(next, absoluteMax))
        onChange(valueMin, next)
      }
    },
    [absoluteMin, absoluteMax, valueMin, valueMax, step, onChange]
  )

  function thumbLeft(v: number): string {
    const frac = (v - absoluteMin) / range
    return `calc(${frac * 100}% + ${HALF_THUMB_PX - frac * THUMB_SIZE_PX}px)`
  }

  const minFrac = (valueMin - absoluteMin) / range
  const maxFrac = (valueMax - absoluteMin) / range
  const highlightStyle = {
    left: `calc(${minFrac * 100}% + ${HALF_THUMB_PX - minFrac * THUMB_SIZE_PX}px)`,
    right: `calc(${(1 - maxFrac) * 100}% + ${maxFrac * THUMB_SIZE_PX - HALF_THUMB_PX}px)`,
  }

  const minActive = valueMin > absoluteMin
  const maxActive = valueMax < absoluteMax

  return (
    <div>
      <div
        ref={outerRef}
        className="relative h-7 cursor-grab select-none active:cursor-grabbing"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        role="group"
      >
        <div className="absolute inset-x-3 top-1/2 h-1.5 -translate-y-1/2 rounded-full bg-slider-track" />
        <div
          className="absolute top-1/2 h-1.5 -translate-y-1/2 rounded-full bg-slider-fill"
          style={highlightStyle}
        />
        <div
          role="slider"
          tabIndex={0}
          aria-label={labelMin}
          aria-valuemin={absoluteMin}
          aria-valuemax={valueMax}
          aria-valuenow={valueMin}
          aria-valuetext={formatValue(valueMin)}
          onKeyDown={(e) => handleKeyDown("min", e)}
          className="absolute top-1/2 h-6 w-6 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 bg-slider-thumb focus:outline-2 focus:outline-offset-2 focus:outline-primary"
          style={{
            left: thumbLeft(valueMin),
            borderColor: minActive
              ? "var(--color-slider-thumb-active)"
              : "var(--color-slider-thumb-inactive)",
          }}
        />
        <div
          role="slider"
          tabIndex={0}
          aria-label={labelMax}
          aria-valuemin={valueMin}
          aria-valuemax={absoluteMax}
          aria-valuenow={valueMax}
          aria-valuetext={formatValue(valueMax)}
          onKeyDown={(e) => handleKeyDown("max", e)}
          className="absolute top-1/2 h-6 w-6 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 bg-slider-thumb focus:outline-2 focus:outline-offset-2 focus:outline-primary"
          style={{
            left: thumbLeft(valueMax),
            borderColor: maxActive
              ? "var(--color-slider-thumb-active)"
              : "var(--color-slider-thumb-inactive)",
          }}
        />
      </div>
      <div className="mt-1 flex justify-between text-xs">
        <span
          className={
            minActive ? "font-medium text-text-secondary" : "text-text-muted"
          }
        >
          {formatValue(valueMin)}
        </span>
        <span
          className={
            maxActive ? "font-medium text-text-secondary" : "text-text-muted"
          }
        >
          {formatValue(valueMax)}
        </span>
      </div>
    </div>
  )
}
