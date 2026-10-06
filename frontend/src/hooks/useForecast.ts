import { useState, useEffect, useCallback, useMemo, useRef } from "react"
import { fetchClient } from "../api/client"
import type {
  ForecastPoint,
  ForecastUnits,
  SunEvent,
  SunIntensity,
} from "../types/forecast"
import { buildForecastTimes, clampForecastStart } from "../lib/time"

const HOUR_MS = 60 * 60 * 1000

/** Default start offset from now, in hours, before the user picks a time. */
const DEFAULT_START_AHEAD_H = 2

/** Coalesces rapid start-time changes (slider drags) into one request. */
const FETCH_DEBOUNCE_MS = 250

const DEFAULT_FORECAST_UNITS: ForecastUnits = {
  temperatureC: "C",
  precipitationRate: "mm/h",
  windSpeedMs: "m/s",
  windDirectionDeg: "deg",
  relativeWindDirectionDeg: "deg",
  solarRadiationWm2: "W/m²",
}

export interface UseForecastResult {
  forecastPoints: ForecastPoint[] | null
  sunEvents: SunEvent[]
  sunIntensity: SunIntensity | null
  forecastLoading: boolean
  forecastStatus: string | null
  forecastAttribution: { text: string; href: string } | undefined
  forecastUnits: ForecastUnits
  forecastTimes: number[] | undefined
  /** Selected forecast start time, snapped to a 15-minute mark. */
  startTime: Date
  speedKmh: number
  estDurationH: number
  /** Sets the start time, snapping and clamping it to the usable range. */
  setStartTime: (t: Date) => void
  setSpeedKmh: (s: number) => void
}

/** Manages forecast state and fetching for a single track. */
export function useForecast(
  uuid: string | undefined,
  totalDistanceM: number | undefined,
  trackDistancesM: number[] | undefined,
  onError: (msg: string) => void
): UseForecastResult {
  const [forecastPoints, setForecastPoints] = useState<ForecastPoint[] | null>(
    null
  )
  const [sunEvents, setSunEvents] = useState<SunEvent[]>([])
  const [sunIntensity, setSunIntensity] = useState<SunIntensity | null>(null)
  const [forecastLoading, setForecastLoading] = useState(false)
  const [forecastStatus, setForecastStatus] = useState<string | null>(null)
  const [forecastAttribution, setForecastAttribution] = useState<
    { text: string; href: string } | undefined
  >(undefined)
  const [forecastUnits, setForecastUnits] = useState<ForecastUnits>(
    DEFAULT_FORECAST_UNITS
  )
  const [startTime, setStartTimeState] = useState<Date>(() => {
    const nowMs = Date.now()
    return new Date(
      clampForecastStart(nowMs + DEFAULT_START_AHEAD_H * HOUR_MS, nowMs)
    )
  })
  const [speedKmh, setSpeedKmh] = useState(28)

  const forecastTimes = useMemo(() => {
    if (!forecastPoints || !trackDistancesM || trackDistancesM.length === 0) {
      return undefined
    }
    return buildForecastTimes(forecastPoints, trackDistancesM)
  }, [forecastPoints, trackDistancesM])

  const estDurationH =
    totalDistanceM && speedKmh > 0 ? totalDistanceM / 1000 / speedKmh : 0

  const setStartTime = useCallback((t: Date) => {
    setStartTimeState(new Date(clampForecastStart(t.getTime(), Date.now())))
  }, [])

  const requestSeq = useRef(0)

  const fetchForecast = useCallback(
    async (start: Date, speed: number) => {
      if (!uuid) return
      const seq = ++requestSeq.current

      setForecastLoading(true)
      setForecastStatus(null)
      try {
        const { data: result, error: apiError } = await fetchClient.GET(
          "/tracks/{uuid}/forecast",
          {
            params: {
              path: { uuid },
              query: {
                startTime: start.toISOString(),
                speedKmh: speed,
              },
            },
          }
        )
        if (apiError) {
          throw new Error(
            (apiError as { msg?: string }).msg ?? "Forecast failed"
          )
        }
        // Drop the response if a newer request superseded it in flight.
        if (seq !== requestSeq.current) {
          return
        }
        if (!result?.points?.length) {
          return
        }
        setForecastStatus(result.forecastStatus ?? null)
        if (result.attribution) {
          setForecastAttribution(result.attribution)
        }
        if (result.units) {
          setForecastUnits(result.units as ForecastUnits)
        }
        setForecastPoints(result.points as ForecastPoint[])
        setSunEvents((result.sunEvents as SunEvent[]) ?? [])
        setSunIntensity((result.sunIntensity as SunIntensity | null) ?? null)
      } catch (err) {
        if (seq !== requestSeq.current) {
          return
        }
        onError((err as Error).message)
      } finally {
        if (seq === requestSeq.current) {
          setForecastLoading(false)
        }
      }
    },
    [uuid, onError]
  )

  const startMs = startTime.getTime()
  useEffect(() => {
    if (!uuid) return
    // Debounced so a slider drag issues a single request once it settles.
    const timer = setTimeout(
      () => fetchForecast(startTime, speedKmh),
      FETCH_DEBOUNCE_MS
    )
    return () => clearTimeout(timer)
  }, [uuid, startMs, speedKmh]) // eslint-disable-line react-hooks/exhaustive-deps

  return {
    forecastPoints,
    sunEvents,
    sunIntensity,
    forecastLoading,
    forecastStatus,
    forecastAttribution,
    forecastUnits,
    forecastTimes,
    startTime,
    speedKmh,
    estDurationH,
    setStartTime,
    setSpeedKmh,
  }
}
